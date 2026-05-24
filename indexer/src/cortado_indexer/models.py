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
