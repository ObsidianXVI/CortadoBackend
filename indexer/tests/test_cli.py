from __future__ import annotations

import io
import json
import sys
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from cortado_indexer.cli import main


class MainCLITest(unittest.TestCase):
    def test_root_indexing_uses_workspace_relative_paths(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            file_path = root / "lib" / "main.py"
            file_path.parent.mkdir(parents=True, exist_ok=True)
            file_path.write_text("def greet():\n    return 'hi'\n", encoding="utf-8")

            stdout = io.StringIO()
            with redirect_stdout(stdout):
                exit_code = main(["--root", str(root)])

        self.assertEqual(exit_code, 0)
        lines = [line for line in stdout.getvalue().splitlines() if line.strip()]
        self.assertGreaterEqual(len(lines), 1)
        payload = json.loads(lines[0])
        self.assertEqual(payload["metadata"]["file"], "lib/main.py")


if __name__ == "__main__":
    unittest.main()
