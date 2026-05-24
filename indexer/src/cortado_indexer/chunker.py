from __future__ import annotations

import importlib
import re
from functools import lru_cache
from pathlib import Path
from typing import Any, Callable

try:
    from tree_sitter import Language, Node, Parser
except ModuleNotFoundError:
    Language = Node = Parser = Any
    TREE_SITTER_AVAILABLE = False
else:
    TREE_SITTER_AVAILABLE = True

from .models import Chunk, ChunkMetadata

FALLBACK_WINDOW_SIZE = 50
FALLBACK_WINDOW_OVERLAP = 10

LanguageChunker = Callable[[str, str], list[Chunk]]

TREE_SITTER_LANGUAGE_MODULES = {
    "python": "tree_sitter_python",
    "javascript": "tree_sitter_javascript",
    "go": "tree_sitter_go",
}


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
    language = detect_language(file_path)
    parser = parser_for_language(language)
    if parser is None:
        return fallback_chunk_file(source, file_path)

    source_bytes = source.encode("utf-8")
    tree = parser.parse(source_bytes)
    return semantic_chunks_for_tree(
        language=language,
        file_path=file_path,
        source=source,
        source_bytes=source_bytes,
        root=tree.root_node,
    )


@lru_cache(maxsize=None)
def parser_for_language(language: str) -> Parser | None:
    if not TREE_SITTER_AVAILABLE:
        return None

    module_name = TREE_SITTER_LANGUAGE_MODULES.get(language)
    if module_name is None:
        return None

    try:
        module = importlib.import_module(module_name)
    except ModuleNotFoundError:
        return None

    parser = Parser(Language(module.language()))
    return parser


def semantic_chunks_for_tree(
    *,
    language: str,
    file_path: str,
    source: str,
    source_bytes: bytes,
    root: Node,
) -> list[Chunk]:
    if language == "python":
        return _python_chunks(file_path, source_bytes, root)
    if language == "javascript":
        return _javascript_chunks(file_path, source_bytes, root)
    if language == "go":
        return _go_chunks(file_path, source_bytes, root)
    return fallback_chunk_file(source, file_path)


def _python_chunks(file_path: str, source_bytes: bytes, root: Node) -> list[Chunk]:
    chunks: list[Chunk] = []
    for child in root.children:
        if child.type == "class_definition":
            class_name = _node_name(child)
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="python",
                    chunker="semantic",
                    symbol_type="class",
                    name=class_name,
                )
            )
            body = child.child_by_field_name("body")
            if body is None:
                continue
            for member in body.children:
                if member.type != "function_definition":
                    continue
                method_name = _node_name(member)
                full_name = (
                    f"{class_name}.{method_name}"
                    if class_name is not None and method_name is not None
                    else method_name
                )
                chunks.append(
                    _chunk_from_node(
                        member,
                        source_bytes,
                        file_path=file_path,
                        language="python",
                        chunker="semantic",
                        symbol_type="method",
                        name=full_name,
                    )
                )
        elif child.type == "function_definition":
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="python",
                    chunker="semantic",
                    symbol_type="function",
                    name=_node_name(child),
                )
            )
    return chunks or fallback_chunk_file(source_bytes.decode("utf-8"), file_path)


def _javascript_chunks(
    file_path: str,
    source_bytes: bytes,
    root: Node,
) -> list[Chunk]:
    chunks: list[Chunk] = []
    for child in root.children:
        if child.type == "class_declaration":
            class_name = _node_name(child)
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="javascript",
                    chunker="semantic",
                    symbol_type="class",
                    name=class_name,
                )
            )
            body = child.child_by_field_name("body")
            if body is None:
                continue
            for member in body.children:
                if member.type != "method_definition":
                    continue
                method_name = _node_name(member)
                full_name = (
                    f"{class_name}.{method_name}"
                    if class_name is not None and method_name is not None
                    else method_name
                )
                chunks.append(
                    _chunk_from_node(
                        member,
                        source_bytes,
                        file_path=file_path,
                        language="javascript",
                        chunker="semantic",
                        symbol_type="method",
                        name=full_name,
                    )
                )
        elif child.type == "function_declaration":
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="javascript",
                    chunker="semantic",
                    symbol_type="function",
                    name=_node_name(child),
                )
            )
        elif child.type == "lexical_declaration":
            declarator = _first_child_of_type(child, "variable_declarator")
            if declarator is None:
                continue
            value = declarator.child_by_field_name("value")
            if value is None or value.type not in {
                "arrow_function",
                "function_expression",
            }:
                continue
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="javascript",
                    chunker="semantic",
                    symbol_type="function",
                    name=_node_name(declarator),
                )
            )
    return chunks or fallback_chunk_file(source_bytes.decode("utf-8"), file_path)


def _go_chunks(file_path: str, source_bytes: bytes, root: Node) -> list[Chunk]:
    chunks: list[Chunk] = []
    for child in root.children:
        if child.type == "type_declaration":
            for spec in child.children:
                if spec.type != "type_spec":
                    continue
                chunks.append(
                    _chunk_from_node(
                        spec,
                        source_bytes,
                        file_path=file_path,
                        language="go",
                        chunker="semantic",
                        symbol_type="type",
                        name=_node_name(spec),
                    )
                )
        elif child.type == "method_declaration":
            receiver = child.child_by_field_name("receiver")
            receiver_name = _go_receiver_name(receiver)
            method_name = _node_name(child)
            full_name = (
                f"{receiver_name}.{method_name}"
                if receiver_name is not None and method_name is not None
                else method_name
            )
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="go",
                    chunker="semantic",
                    symbol_type="method",
                    name=full_name,
                )
            )
        elif child.type == "function_declaration":
            chunks.append(
                _chunk_from_node(
                    child,
                    source_bytes,
                    file_path=file_path,
                    language="go",
                    chunker="semantic",
                    symbol_type="function",
                    name=_node_name(child),
                )
            )
    return chunks or fallback_chunk_file(source_bytes.decode("utf-8"), file_path)


def _chunk_from_node(
    node: Node,
    source_bytes: bytes,
    *,
    file_path: str,
    language: str,
    chunker: str,
    symbol_type: str,
    name: str | None,
) -> Chunk:
    return Chunk(
        text=source_bytes[node.start_byte : node.end_byte].decode("utf-8"),
        metadata=ChunkMetadata(
            file=file_path,
            language=language,
            start_line=node.start_point[0] + 1,
            end_line=node.end_point[0] + 1,
            name=name,
            symbol_type=symbol_type,
            chunker=chunker,
        ),
    )


def _node_name(node: Node) -> str | None:
    name_node = node.child_by_field_name("name")
    if name_node is None:
        return None
    return name_node.text.decode("utf-8")


def _first_child_of_type(node: Node, node_type: str) -> Node | None:
    for child in node.children:
        if child.type == node_type:
            return child
    return None


def _go_receiver_name(receiver: Node | None) -> str | None:
    if receiver is None:
        return None
    matches = re.findall(r"[A-Za-z_][A-Za-z0-9_]*", receiver.text.decode("utf-8"))
    if not matches:
        return None
    return matches[-1]


LANGUAGE_CHUNKERS: dict[str, LanguageChunker] = {
    "dart": tree_sitter_chunk_file,
    "python": tree_sitter_chunk_file,
    "javascript": tree_sitter_chunk_file,
    "go": tree_sitter_chunk_file,
}
