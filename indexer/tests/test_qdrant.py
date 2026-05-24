from __future__ import annotations

import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from cortado_indexer.qdrant import collection_name_for_workspace


class CollectionNameForWorkspaceTest(unittest.TestCase):
    def test_prefixes_workspace_id(self) -> None:
        self.assertEqual(collection_name_for_workspace("ws-123"), "ws-ws-123")

    def test_rejects_empty_workspace_id(self) -> None:
        with self.assertRaisesRegex(ValueError, "workspace_id is required"):
            collection_name_for_workspace("   ")


if __name__ == "__main__":
    unittest.main()
