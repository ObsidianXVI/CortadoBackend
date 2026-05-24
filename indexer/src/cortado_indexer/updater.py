from __future__ import annotations

import json
import logging
import os
import threading
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Protocol, Sequence

from .chunker import chunk_file
from .embedding import (
    DEFAULT_VERTEX_DIMENSIONS,
    VertexAIEmbedder,
    embed_chunks_in_batches,
)
from .models import Chunk, EmbeddedChunk
from .qdrant import (
    QdrantClient,
    collection_name_for_workspace,
    normalize_file_path,
)

FILE_CHANGE_CREATED = "CREATED"
FILE_CHANGE_DELETED = "DELETED"
FILE_CHANGE_MODIFIED = "MODIFIED"
FILE_CHANGE_WINDOW_SECONDS = 5.0
DEFAULT_BATCH_POLL_INTERVAL_SECONDS = 0.25

logger = logging.getLogger(__name__)


class FileChangeEventError(RuntimeError):
    pass


class FileLoader(Protocol):
    def read_text(self, workspace_id: str, file_path: str) -> str: ...


class BatchProcessor(Protocol):
    def process_batch(self, batch: "WorkspaceFileChangeBatch") -> None: ...


@dataclass(frozen=True, slots=True)
class FileChangeEvent:
    workspace_id: str
    path: str
    event_type: str
    checksum: str | None = None
    message_id: str | None = None

    def __post_init__(self) -> None:
        workspace_id = self.workspace_id.strip()
        if not workspace_id:
            raise FileChangeEventError("workspace_id is required")

        if self.event_type not in {
            FILE_CHANGE_CREATED,
            FILE_CHANGE_DELETED,
            FILE_CHANGE_MODIFIED,
        }:
            raise FileChangeEventError(f"unsupported event_type: {self.event_type}")

        object.__setattr__(self, "workspace_id", workspace_id)
        object.__setattr__(self, "path", normalize_workspace_path(self.path))

    @classmethod
    def from_dict(
        cls,
        payload: dict[str, Any],
        *,
        message_id: str | None = None,
    ) -> "FileChangeEvent":
        workspace_id = payload.get("workspace_id") or payload.get("workspaceId")
        path = payload.get("path")
        event_type = payload.get("event_type") or payload.get("eventType")
        checksum = payload.get("checksum")

        if not isinstance(workspace_id, str):
            raise FileChangeEventError("workspace_id is required")
        if not isinstance(path, str):
            raise FileChangeEventError("path is required")
        if not isinstance(event_type, str):
            raise FileChangeEventError("event_type is required")
        if checksum is not None and not isinstance(checksum, str):
            raise FileChangeEventError("checksum must be a string when present")

        return cls(
            workspace_id=workspace_id,
            path=path,
            event_type=event_type.strip().upper(),
            checksum=checksum,
            message_id=message_id,
        )

    def to_dict(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "workspace_id": self.workspace_id,
            "path": self.path,
            "event_type": self.event_type,
        }
        if self.checksum is not None:
            payload["checksum"] = self.checksum
        if self.message_id is not None:
            payload["message_id"] = self.message_id
        return payload


@dataclass(frozen=True, slots=True)
class WorkspaceFileChangeBatch:
    workspace_id: str
    events: tuple[FileChangeEvent, ...]

    def coalesced_events(self) -> list[FileChangeEvent]:
        merged: dict[str, FileChangeEvent] = {}
        for event in self.events:
            current = merged.get(event.path)
            if current is None:
                merged[event.path] = event
                continue
            merged[event.path] = merge_file_change_event(current, event)
        return list(merged.values())


@dataclass(slots=True)
class _PendingWorkspaceBatch:
    events: list[FileChangeEvent]
    deadline_at: float


class WorkspaceEventBatcher:
    def __init__(self, *, window_seconds: float = FILE_CHANGE_WINDOW_SECONDS) -> None:
        if window_seconds <= 0:
            raise ValueError("window_seconds must be positive")
        self.window_seconds = window_seconds
        self._pending: dict[str, _PendingWorkspaceBatch] = {}

    def add(self, event: FileChangeEvent, *, received_at: float | None = None) -> None:
        received_at = time.monotonic() if received_at is None else received_at
        batch = self._pending.get(event.workspace_id)
        if batch is None:
            self._pending[event.workspace_id] = _PendingWorkspaceBatch(
                events=[event],
                deadline_at=received_at + self.window_seconds,
            )
            return
        batch.events.append(event)

    def drain_ready(self, *, now: float | None = None) -> list[WorkspaceFileChangeBatch]:
        now = time.monotonic() if now is None else now
        ready_workspace_ids = sorted(
            workspace_id
            for workspace_id, batch in self._pending.items()
            if batch.deadline_at <= now
        )
        ready_batches: list[WorkspaceFileChangeBatch] = []
        for workspace_id in ready_workspace_ids:
            batch = self._pending.pop(workspace_id)
            ready_batches.append(
                WorkspaceFileChangeBatch(
                    workspace_id=workspace_id,
                    events=tuple(batch.events),
                )
            )
        return ready_batches

    def flush_all(self) -> list[WorkspaceFileChangeBatch]:
        flushed = [
            WorkspaceFileChangeBatch(
                workspace_id=workspace_id,
                events=tuple(batch.events),
            )
            for workspace_id, batch in sorted(self._pending.items())
        ]
        self._pending.clear()
        return flushed


class NoopBatchProcessor:
    def process_batch(self, batch: WorkspaceFileChangeBatch) -> None:
        logger.info(
            "received file change batch for workspace %s with %d event(s)",
            batch.workspace_id,
            len(batch.events),
        )


class LocalWorkspaceFileLoader:
    def __init__(self, root_template: str) -> None:
        root_template = root_template.strip()
        if not root_template:
            raise FileChangeEventError("root_template is required")
        self.root_template = root_template

    @classmethod
    def from_env(cls) -> "LocalWorkspaceFileLoader":
        root_template = os.getenv(
            "CORTADO_WORKSPACE_ROOT_TEMPLATE",
            "/workspace/{workspace_id}",
        )
        return cls(root_template=root_template)

    def read_text(self, workspace_id: str, file_path: str) -> str:
        normalized_path = normalize_workspace_path(file_path)
        workspace_root = Path(self.root_template.format(workspace_id=workspace_id))
        resolved = (workspace_root / normalized_path).resolve()
        root_resolved = workspace_root.resolve()
        if resolved != root_resolved and root_resolved not in resolved.parents:
            raise FileChangeEventError("file_path must stay within the workspace root")
        return resolved.read_text(encoding="utf-8")


class IncrementalIndexProcessor:
    def __init__(
        self,
        *,
        embedder: VertexAIEmbedder,
        file_loader: FileLoader,
        qdrant_client_factory: callable_qdrant_factory | None = None,
        vector_size: int = DEFAULT_VERTEX_DIMENSIONS,
    ) -> None:
        if vector_size <= 0:
            raise FileChangeEventError("vector_size must be positive")
        self.embedder = embedder
        self.file_loader = file_loader
        self.qdrant_client_factory = qdrant_client_factory or default_qdrant_client_factory
        self.vector_size = vector_size

    def process_batch(self, batch: WorkspaceFileChangeBatch) -> None:
        coalesced = batch.coalesced_events()
        if not coalesced:
            return

        collection_name = collection_name_for_workspace(batch.workspace_id)
        qdrant_client = self.qdrant_client_factory(batch.workspace_id)
        upserted = False

        for event in coalesced:
            if event.event_type == FILE_CHANGE_DELETED:
                qdrant_client.delete_by_file_path(collection_name, event.path)

        for event in coalesced:
            if event.event_type == FILE_CHANGE_DELETED:
                continue

            source = self.file_loader.read_text(batch.workspace_id, event.path)
            chunks = chunk_file(source, event.path)
            embedded = embed_chunks_in_batches(chunks, self.embedder)
            if not embedded:
                qdrant_client.delete_by_file_path(collection_name, event.path)
                continue

            if not upserted:
                qdrant_client.ensure_collection(
                    collection_name,
                    vector_size=self.vector_size,
                )
                upserted = True

            qdrant_client.delete_by_file_path(collection_name, event.path)
            qdrant_client.upsert_chunks(collection_name, embedded)


class callable_qdrant_factory(Protocol):
    def __call__(self, workspace_id: str) -> QdrantClient: ...


def default_qdrant_client_factory(workspace_id: str) -> QdrantClient:
    base_url_template = os.getenv(
        "CORTADO_QDRANT_URL_TEMPLATE",
        "http://127.0.0.1:6333",
    )
    base_url = base_url_template.format(workspace_id=workspace_id)
    return QdrantClient(base_url=base_url)


def decode_ingest_request(payload: dict[str, Any]) -> list[FileChangeEvent]:
    message_id: str | None = None
    message_payload: Any = payload

    if "message" in payload:
        message = payload.get("message")
        if not isinstance(message, dict):
            raise FileChangeEventError("message payload must be an object")

        message_id = _optional_string(message.get("messageId"))
        data = message.get("data")
        if not isinstance(data, str) or not data.strip():
            raise FileChangeEventError("message.data is required")

        import base64

        decoded = base64.b64decode(data)
        message_payload = json.loads(decoded.decode("utf-8"))

    events_payload: Sequence[Any]
    if isinstance(message_payload, list):
        events_payload = message_payload
    elif isinstance(message_payload, dict):
        nested_events = message_payload.get("events")
        if nested_events is None:
            events_payload = [message_payload]
        elif isinstance(nested_events, list):
            events_payload = nested_events
        else:
            raise FileChangeEventError("events must be a list")
    else:
        raise FileChangeEventError("ingest payload must decode to an object or list")

    events: list[FileChangeEvent] = []
    for event_payload in events_payload:
        if not isinstance(event_payload, dict):
            raise FileChangeEventError("each event must be an object")
        events.append(FileChangeEvent.from_dict(event_payload, message_id=message_id))
    return events


def merge_file_change_event(current: FileChangeEvent, next_event: FileChangeEvent) -> FileChangeEvent:
    if current.workspace_id != next_event.workspace_id or current.path != next_event.path:
        raise FileChangeEventError("cannot merge file change events for different files")

    if next_event.event_type == FILE_CHANGE_DELETED:
        return next_event
    if current.event_type == FILE_CHANGE_DELETED:
        return next_event
    if current.event_type == FILE_CHANGE_CREATED and next_event.event_type == FILE_CHANGE_MODIFIED:
        return FileChangeEvent(
            workspace_id=current.workspace_id,
            path=current.path,
            event_type=FILE_CHANGE_CREATED,
            checksum=next_event.checksum or current.checksum,
            message_id=next_event.message_id or current.message_id,
        )
    return next_event


def normalize_workspace_path(path: str) -> str:
    normalized = normalize_file_path(path)
    if (
        normalized == "."
        or normalized.startswith("/")
        or normalized.startswith("../")
        or "/../" in normalized
    ):
        raise FileChangeEventError("path must stay within the workspace root")
    return normalized


def run_batch_worker(
    *,
    batcher: WorkspaceEventBatcher,
    processor: BatchProcessor,
    stop_event: threading.Event,
    poll_interval_seconds: float = DEFAULT_BATCH_POLL_INTERVAL_SECONDS,
) -> None:
    while not stop_event.wait(poll_interval_seconds):
        for batch in batcher.drain_ready():
            processor.process_batch(batch)

    for batch in batcher.flush_all():
        processor.process_batch(batch)


def _optional_string(value: Any) -> str | None:
    if isinstance(value, str) and value.strip():
        return value
    return None
