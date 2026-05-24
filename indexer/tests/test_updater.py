from __future__ import annotations

import base64
import json
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from cortado_indexer.models import Chunk, ChunkMetadata, EmbeddedChunk, Embedding
from cortado_indexer.updater import (
    FILE_CHANGE_CREATED,
    FILE_CHANGE_DELETED,
    FILE_CHANGE_MODIFIED,
    FileChangeEvent,
    IncrementalIndexProcessor,
    WorkspaceEventBatcher,
    WorkspaceFileChangeBatch,
    decode_ingest_request,
    merge_file_change_event,
)


def _event(path: str, event_type: str, *, checksum: str | None = None) -> FileChangeEvent:
    return FileChangeEvent(
        workspace_id="ws-123",
        path=path,
        event_type=event_type,
        checksum=checksum,
    )


class FileChangeEventTest(unittest.TestCase):
    def test_from_dict_normalizes_fields(self) -> None:
        event = FileChangeEvent.from_dict(
            {
                "workspaceId": " ws-123 ",
                "path": "./lib\\main.py",
                "eventType": "modified",
                "checksum": "abc123",
            },
            message_id="msg-1",
        )

        self.assertEqual(
            event.to_dict(),
            {
                "workspace_id": "ws-123",
                "path": "lib/main.py",
                "event_type": "MODIFIED",
                "checksum": "abc123",
                "message_id": "msg-1",
            },
        )

    def test_decode_ingest_request_accepts_pubsub_envelope(self) -> None:
        payload = {
            "message": {
                "messageId": "msg-123",
                "data": base64.b64encode(
                    json.dumps(
                        {
                            "events": [
                                {
                                    "workspace_id": "ws-123",
                                    "path": "lib/main.py",
                                    "event_type": "CREATED",
                                }
                            ]
                        }
                    ).encode("utf-8")
                ).decode("utf-8"),
            }
        }

        events = decode_ingest_request(payload)

        self.assertEqual(len(events), 1)
        self.assertEqual(events[0].message_id, "msg-123")
        self.assertEqual(events[0].event_type, FILE_CHANGE_CREATED)


class MergeFileChangeEventTest(unittest.TestCase):
    def test_created_then_modified_stays_created(self) -> None:
        merged = merge_file_change_event(
            _event("lib/main.py", FILE_CHANGE_CREATED, checksum="old"),
            _event("lib/main.py", FILE_CHANGE_MODIFIED, checksum="new"),
        )

        self.assertEqual(merged.event_type, FILE_CHANGE_CREATED)
        self.assertEqual(merged.checksum, "new")

    def test_modified_then_deleted_becomes_deleted(self) -> None:
        merged = merge_file_change_event(
            _event("lib/main.py", FILE_CHANGE_MODIFIED),
            _event("lib/main.py", FILE_CHANGE_DELETED),
        )

        self.assertEqual(merged.event_type, FILE_CHANGE_DELETED)


class WorkspaceEventBatcherTest(unittest.TestCase):
    def test_batches_same_workspace_inside_window(self) -> None:
        batcher = WorkspaceEventBatcher(window_seconds=5.0)

        batcher.add(_event("lib/main.py", FILE_CHANGE_CREATED), received_at=10.0)
        batcher.add(_event("lib/other.py", FILE_CHANGE_MODIFIED), received_at=12.0)

        self.assertEqual(batcher.drain_ready(now=14.9), [])

        ready = batcher.drain_ready(now=15.0)

        self.assertEqual(len(ready), 1)
        self.assertEqual(ready[0].workspace_id, "ws-123")
        self.assertEqual(len(ready[0].events), 2)

    def test_separates_workspaces(self) -> None:
        batcher = WorkspaceEventBatcher(window_seconds=5.0)

        batcher.add(_event("lib/main.py", FILE_CHANGE_CREATED), received_at=1.0)
        batcher.add(
            FileChangeEvent(
                workspace_id="ws-456",
                path="lib/app.py",
                event_type=FILE_CHANGE_MODIFIED,
            ),
            received_at=2.0,
        )

        ready = batcher.drain_ready(now=7.0)

        self.assertEqual([batch.workspace_id for batch in ready], ["ws-123", "ws-456"])

    def test_coalesces_duplicate_paths(self) -> None:
        batch = WorkspaceFileChangeBatch(
            workspace_id="ws-123",
            events=(
                _event("lib/main.py", FILE_CHANGE_CREATED, checksum="a"),
                _event("lib/main.py", FILE_CHANGE_MODIFIED, checksum="b"),
                _event("lib/other.py", FILE_CHANGE_DELETED),
            ),
        )

        coalesced = sorted(batch.coalesced_events(), key=lambda event: event.path)

        self.assertEqual(
            [(event.path, event.event_type, event.checksum) for event in coalesced],
            [
                ("lib/main.py", FILE_CHANGE_CREATED, "b"),
                ("lib/other.py", FILE_CHANGE_DELETED, None),
            ],
        )


class IncrementalIndexProcessorTest(unittest.TestCase):
    def test_process_batch_deletes_then_reindexes_modified_files(self) -> None:
        class Loader:
            def __init__(self) -> None:
                self.calls: list[tuple[str, str]] = []

            def read_text(self, workspace_id: str, file_path: str) -> str:
                self.calls.append((workspace_id, file_path))
                return "def greet():\n    return 'hi'\n"

        class Embedder:
            def __init__(self) -> None:
                self.calls: list[list[str]] = []

            def embed_chunks(self, chunks: list[Chunk]) -> list[EmbeddedChunk]:
                self.calls.append([chunk.metadata.file for chunk in chunks])
                return [
                    EmbeddedChunk(
                        chunk=chunk,
                        embedding=Embedding(
                            values=[0.1, 0.2],
                            provider="vertex-ai",
                            model="text-embedding-004",
                            dimensions=2,
                            task_type="RETRIEVAL_DOCUMENT",
                        ),
                    )
                    for chunk in chunks
                ]

        class QdrantRecorder:
            def __init__(self) -> None:
                self.calls: list[tuple[str, str, object]] = []

            def ensure_collection(self, collection_name: str, *, vector_size: int) -> None:
                self.calls.append(("ensure", collection_name, vector_size))

            def delete_by_file_path(self, collection_name: str, file_path: str) -> None:
                self.calls.append(("delete", collection_name, file_path))

            def upsert_chunks(
                self,
                collection_name: str,
                chunks: list[EmbeddedChunk],
            ) -> None:
                self.calls.append(
                    (
                        "upsert",
                        collection_name,
                        [chunk.chunk.metadata.file for chunk in chunks],
                    )
                )

        loader = Loader()
        embedder = Embedder()
        qdrant = QdrantRecorder()

        processor = IncrementalIndexProcessor(
            embedder=embedder,
            file_loader=loader,
            qdrant_client_factory=lambda _workspace_id: qdrant,
            vector_size=768,
        )

        processor.process_batch(
            WorkspaceFileChangeBatch(
                workspace_id="ws-123",
                events=(
                    _event("lib/main.py", FILE_CHANGE_MODIFIED),
                    _event("lib/deleted.py", FILE_CHANGE_DELETED),
                ),
            )
        )

        self.assertEqual(loader.calls, [("ws-123", "lib/main.py")])
        self.assertEqual(embedder.calls, [["lib/main.py"]])
        self.assertEqual(
            qdrant.calls,
            [
                ("delete", "ws-ws-123", "lib/deleted.py"),
                ("ensure", "ws-ws-123", 768),
                ("delete", "ws-ws-123", "lib/main.py"),
                ("upsert", "ws-ws-123", ["lib/main.py"]),
            ],
        )


if __name__ == "__main__":
    unittest.main()
