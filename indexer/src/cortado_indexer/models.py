from __future__ import annotations

from dataclasses import asdict, dataclass


@dataclass(frozen=True, slots=True)
class ChunkMetadata:
    file: str
    language: str
    start_line: int
    end_line: int
    name: str | None = None
    symbol_type: str | None = None
    chunker: str = "fallback"


@dataclass(frozen=True, slots=True)
class Chunk:
    text: str
    metadata: ChunkMetadata

    def to_dict(self) -> dict[str, object]:
        return {
            "text": self.text,
            "metadata": asdict(self.metadata),
        }


@dataclass(frozen=True, slots=True)
class Embedding:
    values: list[float]
    provider: str
    model: str
    dimensions: int
    task_type: str
    token_count: int | None = None
    truncated: bool | None = None


@dataclass(frozen=True, slots=True)
class EmbeddedChunk:
    chunk: Chunk
    embedding: Embedding

    def to_dict(self) -> dict[str, object]:
        payload = self.chunk.to_dict()
        payload["embedding"] = asdict(self.embedding)
        return payload
