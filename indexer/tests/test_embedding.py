from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from cortado_indexer.embedding import (
    DEFAULT_VERTEX_DIMENSIONS,
    DEFAULT_VERTEX_MODEL,
    DEFAULT_VERTEX_TASK_TYPE,
    EMBEDDING_BATCH_SIZE,
    VertexAIEmbedder,
    batched,
    embed_chunks_in_batches,
)
from cortado_indexer.models import Chunk, ChunkMetadata, EmbeddedChunk, Embedding


def _chunk(name: str) -> Chunk:
    return Chunk(
        text=f"chunk {name}",
        metadata=ChunkMetadata(
            file=f"/workspace/{name}.py",
            language="python",
            start_line=1,
            end_line=2,
            name=name,
            symbol_type="function",
            chunker="semantic",
        ),
    )


class BatchedTest(unittest.TestCase):
    def test_batched_splits_sequences(self) -> None:
        self.assertEqual(batched([1, 2, 3, 4, 5], 2), [[1, 2], [3, 4], [5]])


class VertexAIEmbedderTest(unittest.TestCase):
    def test_embed_chunks_shapes_requests_and_outputs(self) -> None:
        calls: list[tuple[str, dict[str, str], dict[str, object]]] = []
        responses = iter(
            [
                {
                    "predictions": [
                        {
                            "embeddings": {
                                "values": [0.1, 0.2],
                                "statistics": {"token_count": 10, "truncated": False},
                            }
                        },
                        {
                            "embeddings": {
                                "values": [0.3, 0.4],
                                "statistics": {"token_count": 11, "truncated": True},
                            }
                        },
                    ]
                },
                {
                    "predictions": [
                        {
                            "embeddings": {
                                "values": [0.5, 0.6],
                                "statistics": {"token_count": 12, "truncated": False},
                            }
                        }
                    ]
                },
            ]
        )

        def transport(
            url: str,
            headers: dict[str, str],
            body: bytes,
        ) -> dict[str, object]:
            calls.append((url, headers, json.loads(body.decode("utf-8"))))
            return next(responses)

        embedder = VertexAIEmbedder(
            project_id="cortado-ide",
            dimensions=DEFAULT_VERTEX_DIMENSIONS,
            model=DEFAULT_VERTEX_MODEL,
            request_batch_size=2,
            token_provider=lambda: "token-123",
            transport=transport,
        )

        embedded = embedder.embed_chunks([_chunk("one"), _chunk("two"), _chunk("three")])

        self.assertEqual(len(calls), 2)
        self.assertEqual(
            calls[0][0],
            "https://us-central1-aiplatform.googleapis.com/v1/projects/"
            "cortado-ide/locations/us-central1/publishers/google/models/"
            "text-embedding-004:predict",
        )
        self.assertEqual(calls[0][1]["Authorization"], "Bearer token-123")
        self.assertEqual(
            calls[0][2]["parameters"],
            {"autoTruncate": True, "outputDimensionality": DEFAULT_VERTEX_DIMENSIONS},
        )
        self.assertEqual(
            calls[0][2]["instances"],
            [
                {"content": "chunk one", "task_type": DEFAULT_VERTEX_TASK_TYPE},
                {"content": "chunk two", "task_type": DEFAULT_VERTEX_TASK_TYPE},
            ],
        )
        self.assertEqual(len(embedded), 3)
        self.assertEqual(embedded[0].embedding.provider, "vertex-ai")
        self.assertEqual(embedded[1].embedding.truncated, True)
        self.assertEqual(embedded[2].embedding.token_count, 12)
        self.assertEqual(embedded[2].chunk.metadata.name, "three")


class EmbedChunksInBatchesTest(unittest.TestCase):
    def test_embed_chunks_in_batches_uses_pipeline_batch_size(self) -> None:
        chunks = [_chunk(str(index)) for index in range(EMBEDDING_BATCH_SIZE + 1)]

        class RecordingEmbedder:
            def __init__(self) -> None:
                self.batch_sizes: list[int] = []

            def embed_chunks(self, batch: list[Chunk]) -> list[EmbeddedChunk]:
                self.batch_sizes.append(len(batch))
                return [
                    EmbeddedChunk(
                        chunk=chunk,
                        embedding=Embedding(
                            values=[1.0],
                            provider="vertex-ai",
                            model=DEFAULT_VERTEX_MODEL,
                            dimensions=1,
                            task_type=DEFAULT_VERTEX_TASK_TYPE,
                        ),
                    )
                    for chunk in batch
                ]

        embedder = RecordingEmbedder()

        embedded = embed_chunks_in_batches(chunks, embedder, batch_size=EMBEDDING_BATCH_SIZE)

        self.assertEqual(embedder.batch_sizes, [EMBEDDING_BATCH_SIZE, 1])
        self.assertEqual(len(embedded), len(chunks))


if __name__ == "__main__":
    unittest.main()
