from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from cortado_indexer.chunker import (
    FALLBACK_WINDOW_OVERLAP,
    FALLBACK_WINDOW_SIZE,
    chunk_file,
    detect_language,
    parser_for_language,
)


class DetectLanguageTest(unittest.TestCase):
    def test_detects_supported_languages(self) -> None:
        self.assertEqual(detect_language("lib/main.dart"), "dart")
        self.assertEqual(detect_language("pkg/mod.py"), "python")
        self.assertEqual(detect_language("web/app.ts"), "javascript")
        self.assertEqual(detect_language("cmd/server.go"), "go")
        self.assertEqual(detect_language("README.md"), "plain")


class FallbackChunkerTest(unittest.TestCase):
    def test_fallback_chunker_uses_overlap_windows(self) -> None:
        lines = [f"line {index}" for index in range(1, 121)]
        chunks = chunk_file("\n".join(lines), "/workspace/lib/main.dart")

        self.assertEqual(len(chunks), 3)
        self.assertEqual(chunks[0].metadata.start_line, 1)
        self.assertEqual(chunks[0].metadata.end_line, FALLBACK_WINDOW_SIZE)
        self.assertEqual(
            chunks[1].metadata.start_line,
            FALLBACK_WINDOW_SIZE - FALLBACK_WINDOW_OVERLAP + 1,
        )
        self.assertEqual(chunks[-1].metadata.end_line, 120)
        self.assertEqual(chunks[0].metadata.language, "dart")

    def test_empty_files_produce_no_chunks(self) -> None:
        self.assertEqual(chunk_file("", "/workspace/lib/main.dart"), [])


def _semantic_parser_available(language: str) -> bool:
    return parser_for_language(language) is not None


class SemanticPythonChunkerTest(unittest.TestCase):
    @unittest.skipUnless(
        _semantic_parser_available("python"),
        "python tree-sitter grammar is not installed",
    )
    def test_extracts_class_method_and_function_chunks(self) -> None:
        chunks = chunk_file(
            textwrap.dedent(
                """\
                class Foo:
                    def method(self):
                        return 1

                def top():
                    return 2
                """
            ),
            "/workspace/pkg/example.py",
        )

        self.assertEqual(
            [(chunk.metadata.name, chunk.metadata.symbol_type) for chunk in chunks],
            [
                ("Foo", "class"),
                ("Foo.method", "method"),
                ("top", "function"),
            ],
        )
        self.assertTrue(all(chunk.metadata.chunker == "semantic" for chunk in chunks))
        self.assertEqual((chunks[0].metadata.start_line, chunks[0].metadata.end_line), (1, 3))
        self.assertEqual((chunks[1].metadata.start_line, chunks[1].metadata.end_line), (2, 3))
        self.assertEqual((chunks[2].metadata.start_line, chunks[2].metadata.end_line), (5, 6))


class SemanticJavaScriptChunkerTest(unittest.TestCase):
    @unittest.skipUnless(
        _semantic_parser_available("javascript"),
        "javascript tree-sitter grammar is not installed",
    )
    def test_extracts_class_function_and_arrow_chunks(self) -> None:
        chunks = chunk_file(
            textwrap.dedent(
                """\
                class Foo {
                  method() {
                    return 1;
                  }
                }

                function top() {
                  return 2;
                }

                const arrow = () => {
                  return 3;
                };
                """
            ),
            "/workspace/web/example.ts",
        )

        self.assertEqual(
            [(chunk.metadata.name, chunk.metadata.symbol_type) for chunk in chunks],
            [
                ("Foo", "class"),
                ("Foo.method", "method"),
                ("top", "function"),
                ("arrow", "function"),
            ],
        )
        self.assertTrue(all(chunk.metadata.chunker == "semantic" for chunk in chunks))
        self.assertEqual((chunks[0].metadata.start_line, chunks[0].metadata.end_line), (1, 5))
        self.assertEqual((chunks[1].metadata.start_line, chunks[1].metadata.end_line), (2, 4))
        self.assertEqual((chunks[2].metadata.start_line, chunks[2].metadata.end_line), (7, 9))
        self.assertEqual((chunks[3].metadata.start_line, chunks[3].metadata.end_line), (11, 13))


class SemanticGoChunkerTest(unittest.TestCase):
    @unittest.skipUnless(
        _semantic_parser_available("go"),
        "go tree-sitter grammar is not installed",
    )
    def test_extracts_type_method_and_function_chunks(self) -> None:
        chunks = chunk_file(
            textwrap.dedent(
                """\
                package sample

                type Foo struct{}

                func (f *Foo) Method() {
                    return
                }

                func Top() {}
                """
            ),
            "/workspace/cmd/example.go",
        )

        self.assertEqual(
            [(chunk.metadata.name, chunk.metadata.symbol_type) for chunk in chunks],
            [
                ("Foo", "type"),
                ("Foo.Method", "method"),
                ("Top", "function"),
            ],
        )
        self.assertTrue(all(chunk.metadata.chunker == "semantic" for chunk in chunks))
        self.assertEqual((chunks[0].metadata.start_line, chunks[0].metadata.end_line), (3, 3))
        self.assertEqual((chunks[1].metadata.start_line, chunks[1].metadata.end_line), (5, 7))
        self.assertEqual((chunks[2].metadata.start_line, chunks[2].metadata.end_line), (9, 9))


class CliTest(unittest.TestCase):
    def test_cli_emits_newline_delimited_json(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            file_path = Path(temp_dir) / "main.dart"
            file_path.write_text(
                textwrap.dedent(
                    """
                    void main() {
                      print("hello");
                    }
                    """
                ).strip(),
                encoding="utf-8",
            )
            result = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "cortado_indexer",
                    "--file",
                    str(file_path),
                ],
                cwd=REPO_ROOT,
                env={
                    "PYTHONPATH": str(REPO_ROOT / "src"),
                },
                capture_output=True,
                check=True,
                text=True,
            )

        output_lines = [line for line in result.stdout.splitlines() if line]
        self.assertEqual(len(output_lines), 1)
        payload = json.loads(output_lines[0])
        self.assertEqual(payload["metadata"]["language"], "dart")
        self.assertEqual(payload["metadata"]["file"], str(file_path))


if __name__ == "__main__":
    unittest.main()
