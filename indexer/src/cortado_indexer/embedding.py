from __future__ import annotations

import json
import os
from typing import Any, Callable, Sequence, TypeVar
from urllib import error as urllib_error
from urllib import request as urllib_request

try:
    import google.auth
    from google.auth.transport.requests import Request as GoogleAuthRequest
except ImportError:
    google = None
    GoogleAuthRequest = Any
    GOOGLE_AUTH_AVAILABLE = False
else:
    GOOGLE_AUTH_AVAILABLE = True

from .models import Chunk, EmbeddedChunk, Embedding

DEFAULT_VERTEX_DIMENSIONS = 768
DEFAULT_VERTEX_LOCATION = "us-central1"
DEFAULT_VERTEX_MODEL = "text-embedding-004"
DEFAULT_VERTEX_PROVIDER = "vertex-ai"
DEFAULT_VERTEX_TASK_TYPE = "RETRIEVAL_DOCUMENT"
EMBEDDING_BATCH_SIZE = 100
VERTEX_REQUEST_BATCH_SIZE = 5
VERTEX_SCOPE = "https://www.googleapis.com/auth/cloud-platform"

Transport = Callable[[str, dict[str, str], bytes], dict[str, Any]]
TokenProvider = Callable[[], str]
T = TypeVar("T")


class EmbeddingConfigError(RuntimeError):
    pass


class EmbeddingRequestError(RuntimeError):
    pass


def batched(items: Sequence[T], batch_size: int) -> list[Sequence[T]]:
    if batch_size <= 0:
        raise ValueError("batch_size must be positive")
    return [items[index : index + batch_size] for index in range(0, len(items), batch_size)]


class VertexAIEmbedder:
    def __init__(
        self,
        *,
        project_id: str,
        location: str = DEFAULT_VERTEX_LOCATION,
        model: str = DEFAULT_VERTEX_MODEL,
        dimensions: int = DEFAULT_VERTEX_DIMENSIONS,
        task_type: str = DEFAULT_VERTEX_TASK_TYPE,
        request_batch_size: int = VERTEX_REQUEST_BATCH_SIZE,
        token_provider: TokenProvider | None = None,
        transport: Transport | None = None,
    ) -> None:
        if not project_id:
            raise EmbeddingConfigError("project_id is required")
        if dimensions <= 0:
            raise EmbeddingConfigError("dimensions must be positive")
        if request_batch_size <= 0:
            raise EmbeddingConfigError("request_batch_size must be positive")

        self.project_id = project_id
        self.location = location
        self.model = model
        self.dimensions = dimensions
        self.task_type = task_type
        self.request_batch_size = request_batch_size
        self._token_provider = token_provider or self._default_token_provider()
        self._transport = transport or self._default_transport

    @classmethod
    def from_env(cls) -> VertexAIEmbedder:
        project_id = os.getenv("CORTADO_VERTEX_PROJECT_ID") or os.getenv(
            "GOOGLE_CLOUD_PROJECT"
        )
        if not project_id:
            raise EmbeddingConfigError(
                "set CORTADO_VERTEX_PROJECT_ID or GOOGLE_CLOUD_PROJECT to use --embed"
            )

        return cls(
            project_id=project_id,
            location=os.getenv("CORTADO_VERTEX_LOCATION", DEFAULT_VERTEX_LOCATION),
            model=os.getenv("CORTADO_VERTEX_MODEL", DEFAULT_VERTEX_MODEL),
            dimensions=int(
                os.getenv("CORTADO_VERTEX_DIMENSIONS", str(DEFAULT_VERTEX_DIMENSIONS))
            ),
            task_type=os.getenv("CORTADO_VERTEX_TASK_TYPE", DEFAULT_VERTEX_TASK_TYPE),
            request_batch_size=int(
                os.getenv(
                    "CORTADO_VERTEX_REQUEST_BATCH_SIZE",
                    str(VERTEX_REQUEST_BATCH_SIZE),
                )
            ),
        )

    def embed_chunks(self, chunks: Sequence[Chunk]) -> list[EmbeddedChunk]:
        embedded: list[EmbeddedChunk] = []
        for batch in batched(chunks, self.request_batch_size):
            response = self._predict(batch)
            predictions = response.get("predictions")
            if not isinstance(predictions, list) or len(predictions) != len(batch):
                raise EmbeddingRequestError(
                    f"expected {len(batch)} predictions, received {predictions!r}"
                )

            for chunk, prediction in zip(batch, predictions):
                embedded.append(self._embedded_chunk(chunk, prediction))
        return embedded

    def endpoint(self) -> str:
        return (
            f"https://{self.location}-aiplatform.googleapis.com/v1/projects/"
            f"{self.project_id}/locations/{self.location}/publishers/google/models/"
            f"{self.model}:predict"
        )

    def _predict(self, chunks: Sequence[Chunk]) -> dict[str, Any]:
        payload = {
            "instances": [
                {
                    "content": chunk.text,
                    "task_type": self.task_type,
                }
                for chunk in chunks
            ],
            "parameters": {
                "autoTruncate": True,
                "outputDimensionality": self.dimensions,
            },
        }

        body = json.dumps(payload, sort_keys=True).encode("utf-8")
        headers = {
            "Authorization": f"Bearer {self._token_provider()}",
            "Content-Type": "application/json; charset=utf-8",
        }
        return self._transport(self.endpoint(), headers, body)

    def _embedded_chunk(self, chunk: Chunk, prediction: Any) -> EmbeddedChunk:
        if not isinstance(prediction, dict):
            raise EmbeddingRequestError(f"unexpected prediction payload: {prediction!r}")
        embedding_payload = prediction.get("embeddings")
        if not isinstance(embedding_payload, dict):
            raise EmbeddingRequestError(
                f"missing embeddings payload for chunk {chunk.metadata.file}"
            )
        values = embedding_payload.get("values")
        if not isinstance(values, list):
            raise EmbeddingRequestError(
                f"missing embedding values for chunk {chunk.metadata.file}"
            )
        statistics = embedding_payload.get("statistics") or {}
        if not isinstance(statistics, dict):
            statistics = {}

        return EmbeddedChunk(
            chunk=chunk,
            embedding=Embedding(
                values=[float(value) for value in values],
                provider=DEFAULT_VERTEX_PROVIDER,
                model=self.model,
                dimensions=self.dimensions,
                task_type=self.task_type,
                token_count=_maybe_int(statistics.get("token_count")),
                truncated=_maybe_bool(statistics.get("truncated")),
            ),
        )

    def _default_token_provider(self) -> TokenProvider:
        if not GOOGLE_AUTH_AVAILABLE:
            raise EmbeddingConfigError(
                "google-auth is required to request Vertex AI embeddings"
            )

        credentials, _ = google.auth.default(scopes=[VERTEX_SCOPE])
        auth_request = GoogleAuthRequest()

        def provide_token() -> str:
            if not credentials.valid or credentials.token is None:
                credentials.refresh(auth_request)
            if credentials.token is None:
                raise EmbeddingRequestError("Vertex AI credentials did not yield a token")
            return credentials.token

        return provide_token

    @staticmethod
    def _default_transport(
        url: str,
        headers: dict[str, str],
        body: bytes,
    ) -> dict[str, Any]:
        request = urllib_request.Request(
            url=url,
            data=body,
            headers=headers,
            method="POST",
        )
        try:
            with urllib_request.urlopen(request, timeout=30) as response:
                return json.loads(response.read().decode("utf-8"))
        except urllib_error.HTTPError as err:
            detail = err.read().decode("utf-8", errors="replace")
            raise EmbeddingRequestError(
                f"Vertex AI embedding request failed with status {err.code}: {detail}"
            ) from err
        except urllib_error.URLError as err:
            raise EmbeddingRequestError(
                f"Vertex AI embedding request failed: {err.reason}"
            ) from err


def embed_chunks_in_batches(
    chunks: Sequence[Chunk],
    embedder: VertexAIEmbedder,
    *,
    batch_size: int = EMBEDDING_BATCH_SIZE,
) -> list[EmbeddedChunk]:
    embedded: list[EmbeddedChunk] = []
    for batch in batched(chunks, batch_size):
        embedded.extend(embedder.embed_chunks(batch))
    return embedded


def _maybe_bool(value: Any) -> bool | None:
    if isinstance(value, bool):
        return value
    return None


def _maybe_int(value: Any) -> int | None:
    if isinstance(value, int):
        return value
    return None
