from __future__ import annotations

import argparse
import json
from pathlib import Path

from .chunker import chunk_file


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="cortado-indexer",
        description="Chunk source files for Cortado indexing jobs.",
    )
    parser.add_argument(
        "--root",
        type=Path,
        help="Recursively chunk supported source files under this workspace root.",
    )
    parser.add_argument(
        "--file",
        action="append",
        type=Path,
        default=[],
        help="Chunk a specific file. Can be passed multiple times.",
    )
    parser.add_argument(
        "--embed",
        action="store_true",
        help="Request Vertex AI embeddings for each emitted chunk.",
    )
    return parser


def iter_source_files(root: Path) -> list[Path]:
    return sorted(
        path
        for path in root.rglob("*")
        if path.is_file()
        and path.suffix.lower() in {".dart", ".py", ".js", ".jsx", ".ts", ".tsx", ".go"}
    )


def build_targets(root: Path | None, files: list[Path]) -> list[tuple[Path, str]]:
    targets = [(path, str(path)) for path in files]
    if root is None:
        return targets

    root = root.resolve()
    targets.extend(
        (path, path.resolve().relative_to(root).as_posix()) for path in iter_source_files(root)
    )
    return targets


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    targets = build_targets(args.root, list(args.file))

    if not targets:
        parser.error("pass --root or at least one --file")

    for path, logical_path in targets:
        source = path.read_text(encoding="utf-8")
        chunks = chunk_file(source, logical_path)
        if args.embed:
            from .embedding import VertexAIEmbedder, embed_chunks_in_batches

            embedder = VertexAIEmbedder.from_env()
            for chunk in embed_chunks_in_batches(chunks, embedder):
                print(json.dumps(chunk.to_dict(), sort_keys=True))
            continue
        for chunk in chunks:
            print(json.dumps(chunk.to_dict(), sort_keys=True))
    return 0
