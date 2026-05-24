from __future__ import annotations

QDRANT_COLLECTION_PREFIX = "ws-"


def collection_name_for_workspace(workspace_id: str) -> str:
    workspace_id = workspace_id.strip()
    if not workspace_id:
        raise ValueError("workspace_id is required")
    return f"{QDRANT_COLLECTION_PREFIX}{workspace_id}"
