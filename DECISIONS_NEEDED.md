# Decisions Needed

- Task 5.1.2 embedding provider selection:
  Choose whether the first embedding pipeline implementation should target `text-embedding-004` on Vertex AI or `voyage-code-3` on Voyage AI.
  Context: the feature spec allows either, but this choice fixes vector dimensionality, dependency surface, credential flow, and the first indexer/updater client integration. If no product reason points to Voyage, the lowest-friction default is Vertex AI because the rest of the stack already runs on GCP.
