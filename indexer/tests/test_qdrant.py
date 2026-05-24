from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from cortado_indexer.models import Chunk, ChunkMetadata, EmbeddedChunk, Embedding
from cortado_indexer.qdrant import (
    QdrantClient,
    collection_name_for_workspace,
    normalize_file_path,
    point_for_chunk,
)


def _embedded_chunk() -> EmbeddedChunk:
    return EmbeddedChunk(
        chunk=Chunk(
            text="def greet():\n    return 'hi'\n",
            metadata=ChunkMetadata(
                file="lib/main.py",
                language="python",
                start_line=1,
                end_line=2,
                name="greet",
                symbol_type="function",
                chunker="semantic",
            ),
        ),
        embedding=Embedding(
            values=[0.1, 0.2, 0.3],
            provider="vertex-ai",
            model="text-embedding-004",
            dimensions=3,
            task_type="RETRIEVAL_DOCUMENT",
            token_count=4,
            truncated=False,
        ),
    )


class CollectionNameForWorkspaceTest(unittest.TestCase):
    def test_prefixes_workspace_id(self) -> None:
        self.assertEqual(collection_name_for_workspace("ws-123"), "ws-ws-123")

    def test_rejects_empty_workspace_id(self) -> None:
        with self.assertRaisesRegex(ValueError, "workspace_id is required"):
            collection_name_for_workspace("   ")


class NormalizeFilePathTest(unittest.TestCase):
    def test_normalizes_separators_and_dot_prefix(self) -> None:
        self.assertEqual(normalize_file_path("./lib\\main.py"), "lib/main.py")

    def test_rejects_empty_path(self) -> None:
        with self.assertRaisesRegex(ValueError, "file_path is required"):
            normalize_file_path(" ")


class PointForChunkTest(unittest.TestCase):
    def test_shapes_payload_without_embedding_values_duplication(self) -> None:
        point = point_for_chunk(_embedded_chunk())

        self.assertIsInstance(point["id"], str)
        self.assertEqual(point["vector"], [0.1, 0.2, 0.3])
        self.assertEqual(point["payload"]["metadata"]["file"], "lib/main.py")
        self.assertNotIn("values", point["payload"]["embedding"])


class QdrantClientTest(unittest.TestCase):
    def test_ensure_collection_shapes_request(self) -> None:
        calls: list[tuple[str, str, dict[str, str], dict[str, object]]] = []

        def transport(
            url: str,
            method: str,
            headers: dict[str, str],
            body: bytes | None,
        ) -> dict[str, object]:
            self.assertIsNotNone(body)
            calls.append((url, method, headers, json.loads(body.decode("utf-8"))))
            return {"status": "ok"}

        client = QdrantClient(base_url="http://qdrant:6333", transport=transport)

        response = client.ensure_collection("ws-123", vector_size=768)

        self.assertEqual(response, {"status": "ok"})
        self.assertEqual(
            calls,
            [
                (
                    "http://qdrant:6333/collections/ws-123",
                    "PUT",
                    {
                        "Accept": "application/json",
                        "Content-Type": "application/json; charset=utf-8",
                    },
                    {"vectors": {"distance": "Cosine", "size": 768}},
                )
            ],
        )

    def test_upsert_chunks_shapes_request(self) -> None:
        calls: list[tuple[str, str, dict[str, str], dict[str, object]]] = []

        def transport(
            url: str,
            method: str,
            headers: dict[str, str],
            body: bytes | None,
        ) -> dict[str, object]:
            self.assertIsNotNone(body)
            calls.append((url, method, headers, json.loads(body.decode("utf-8"))))
            return {"result": {"status": "acknowledged"}}

        client = QdrantClient(base_url="http://qdrant:6333", transport=transport)

        response = client.upsert_chunks("ws-123", [_embedded_chunk()])

        self.assertEqual(response, {"result": {"status": "acknowledged"}})
        self.assertEqual(calls[0][0], "http://qdrant:6333/collections/ws-123/points?wait=true")
        self.assertEqual(calls[0][1], "PUT")
        self.assertEqual(len(calls[0][3]["points"]), 1)
        self.assertEqual(calls[0][3]["points"][0]["payload"]["metadata"]["file"], "lib/main.py")

    def test_delete_by_file_path_shapes_request(self) -> None:
        calls: list[tuple[str, str, dict[str, str], dict[str, object]]] = []

        def transport(
            url: str,
            method: str,
            headers: dict[str, str],
            body: bytes | None,
        ) -> dict[str, object]:
            self.assertIsNotNone(body)
            calls.append((url, method, headers, json.loads(body.decode("utf-8"))))
            return {"status": "ok"}

        client = QdrantClient(base_url="http://qdrant:6333", transport=transport)

        response = client.delete_by_file_path("ws-123", "./lib/main.py")

        self.assertEqual(response, {"status": "ok"})
        self.assertEqual(
            calls,
            [
                (
                    "http://qdrant:6333/collections/ws-123/points/delete?wait=true",
                    "POST",
                    {
                        "Accept": "application/json",
                        "Content-Type": "application/json; charset=utf-8",
                    },
                    {
                        "filter": {
                            "must": [
                                {
                                    "key": "metadata.file",
                                    "match": {"value": "lib/main.py"},
                                }
                            ]
                        }
                    },
                )
            ],
        )


if __name__ == "__main__":
    unittest.main()
