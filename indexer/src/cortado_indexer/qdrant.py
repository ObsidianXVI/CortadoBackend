from __future__ import annotations

import hashlib
import json
import posixpath
from dataclasses import asdict
from typing import Any, Callable, Sequence
from urllib import error as urllib_error
from urllib import parse as urllib_parse
from urllib import request as urllib_request

from .models import Chunk, EmbeddedChunk

QDRANT_COLLECTION_PREFIX = "ws-"
QDRANT_DEFAULT_DISTANCE = "Cosine"
QDRANT_DEFAULT_TIMEOUT_SECONDS = 30
QDRANT_DEFAULT_URL = "http://127.0.0.1:6333"

Transport = Callable[[str, str, dict[str, str], bytes | None], dict[str, Any]]


class QdrantConfigError(RuntimeError):
    pass


class QdrantRequestError(RuntimeError):
    pass


class QdrantClient:
    def __init__(
        self,
        *,
        base_url: str = QDRANT_DEFAULT_URL,
        timeout_seconds: int = QDRANT_DEFAULT_TIMEOUT_SECONDS,
        transport: Transport | None = None,
    ) -> None:
        base_url = base_url.rstrip("/")
        if not base_url:
            raise QdrantConfigError("base_url is required")
        if timeout_seconds <= 0:
            raise QdrantConfigError("timeout_seconds must be positive")

        self.base_url = base_url
        self.timeout_seconds = timeout_seconds
        self._transport = transport or self._default_transport

    def ensure_collection(
        self,
        collection_name: str,
        *,
        vector_size: int,
        distance: str = QDRANT_DEFAULT_DISTANCE,
    ) -> dict[str, Any]:
        if vector_size <= 0:
            raise QdrantConfigError("vector_size must be positive")

        return self._request(
            "PUT",
            f"/collections/{urllib_parse.quote(collection_name, safe='')}",
            {
                "vectors": {
                    "size": vector_size,
                    "distance": distance,
                }
            },
        )

    def upsert_chunks(
        self,
        collection_name: str,
        chunks: Sequence[EmbeddedChunk],
    ) -> dict[str, Any] | None:
        if not chunks:
            return None

        return self._request(
            "PUT",
            f"/collections/{urllib_parse.quote(collection_name, safe='')}/points?wait=true",
            {"points": [point_for_chunk(chunk) for chunk in chunks]},
        )

    def delete_by_file_path(
        self,
        collection_name: str,
        file_path: str,
    ) -> dict[str, Any]:
        file_path = normalize_file_path(file_path)
        return self._request(
            "POST",
            f"/collections/{urllib_parse.quote(collection_name, safe='')}/points/delete?wait=true",
            {
                "filter": {
                    "must": [
                        {
                            "key": "metadata.file",
                            "match": {"value": file_path},
                        }
                    ]
                }
            },
        )

    def _request(
        self,
        method: str,
        path: str,
        payload: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        body = None
        headers = {"Accept": "application/json"}
        if payload is not None:
            body = json.dumps(payload, sort_keys=True).encode("utf-8")
            headers["Content-Type"] = "application/json; charset=utf-8"

        return self._transport(self.base_url + path, method, headers, body)

    def _default_transport(
        self,
        url: str,
        method: str,
        headers: dict[str, str],
        body: bytes | None,
    ) -> dict[str, Any]:
        request = urllib_request.Request(
            url=url,
            data=body,
            headers=headers,
            method=method,
        )
        try:
            with urllib_request.urlopen(request, timeout=self.timeout_seconds) as response:
                raw = response.read().decode("utf-8")
                if not raw:
                    return {}
                return json.loads(raw)
        except urllib_error.HTTPError as err:
            detail = err.read().decode("utf-8", errors="replace")
            raise QdrantRequestError(
                f"Qdrant request failed with status {err.code}: {detail}"
            ) from err
        except urllib_error.URLError as err:
            raise QdrantRequestError(f"Qdrant request failed: {err.reason}") from err


def collection_name_for_workspace(workspace_id: str) -> str:
    workspace_id = workspace_id.strip()
    if not workspace_id:
        raise ValueError("workspace_id is required")
    return f"{QDRANT_COLLECTION_PREFIX}{workspace_id}"


def normalize_file_path(file_path: str) -> str:
    normalized = posixpath.normpath(file_path.strip().replace("\\", "/"))
    while normalized.startswith("./"):
        normalized = normalized[2:]
    if normalized in {"", "."}:
        raise ValueError("file_path is required")
    return normalized


def point_for_chunk(chunk: EmbeddedChunk) -> dict[str, Any]:
    return {
        "id": point_id_for_chunk(chunk.chunk),
        "payload": {
            "text": chunk.chunk.text,
            "metadata": asdict(chunk.chunk.metadata),
            "embedding": {
                "provider": chunk.embedding.provider,
                "model": chunk.embedding.model,
                "dimensions": chunk.embedding.dimensions,
                "task_type": chunk.embedding.task_type,
                "token_count": chunk.embedding.token_count,
                "truncated": chunk.embedding.truncated,
            },
        },
        "vector": chunk.embedding.values,
    }


def point_id_for_chunk(chunk: Chunk) -> str:
    payload = json.dumps(
        {
            "file": chunk.metadata.file,
            "start_line": chunk.metadata.start_line,
            "end_line": chunk.metadata.end_line,
            "name": chunk.metadata.name,
            "symbol_type": chunk.metadata.symbol_type,
            "chunker": chunk.metadata.chunker,
        },
        sort_keys=True,
        separators=(",", ":"),
    )
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()
