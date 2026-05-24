from __future__ import annotations

from pathlib import Path
from typing import Callable

from .models import Chunk, ChunkMetadata

FALLBACK_WINDOW_SIZE = 50
FALLBACK_WINDOW_OVERLAP = 10

LanguageChunker = Callable[[str, str], list[Chunk]]


def detect_language(file_path: str) -> str:
    suffix = Path(file_path).suffix.lower()
    return {
        ".dart": "dart",
        ".py": "python",
        ".js": "javascript",
        ".jsx": "javascript",
        ".ts": "javascript",
        ".tsx": "javascript",
        ".go": "go",
    }.get(suffix, "plain")


def chunk_file(source: str, file_path: str) -> list[Chunk]:
    language = detect_language(file_path)
    chunker = LANGUAGE_CHUNKERS.get(language, fallback_chunk_file)
    return chunker(source, file_path)


def fallback_chunk_file(source: str, file_path: str) -> list[Chunk]:
    language = detect_language(file_path)
    lines = source.splitlines()
    if not lines:
        return []

    step = max(1, FALLBACK_WINDOW_SIZE - FALLBACK_WINDOW_OVERLAP)
    chunks: list[Chunk] = []
    for start_index in range(0, len(lines), step):
        end_index = min(start_index + FALLBACK_WINDOW_SIZE, len(lines))
        chunk_lines = lines[start_index:end_index]
        if not chunk_lines:
            continue
        chunks.append(
            Chunk(
                text="\n".join(chunk_lines),
                metadata=ChunkMetadata(
                    file=file_path,
                    language=language,
                    start_line=start_index + 1,
                    end_line=end_index,
                ),
            )
        )
        if end_index >= len(lines):
            break
    return chunks


def tree_sitter_chunk_file(source: str, file_path: str) -> list[Chunk]:
    # The parser registry lives behind a single function so the scaffold can
    # start with a stable fallback path while native grammar loading is wired
    # in via the Docker build flow.
    return fallback_chunk_file(source, file_path)


LANGUAGE_CHUNKERS: dict[str, LanguageChunker] = {
    "dart": tree_sitter_chunk_file,
    "python": tree_sitter_chunk_file,
    "javascript": tree_sitter_chunk_file,
    "go": tree_sitter_chunk_file,
}
