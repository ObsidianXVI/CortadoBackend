# Cortado Indexer

`indexer/` is the Python job scaffold for v0.5 Feature 5.1.

Current scope:
- chunk source files for indexing
- prefer semantic chunking when a parser is wired for the language
- fall back to overlapping line windows when no parser is available yet

The initial scaffold keeps the dependency surface conservative:
- `tree-sitter==0.22.0` is pinned per the release spec
- language-specific parser wiring is intentionally isolated behind a registry
- the Docker image is expected to build any native tree-sitter artifacts during image build, not at runtime

## Local usage

Run the CLI against a workspace directory:

```bash
cd indexer
PYTHONPATH=src python3 -m cortado_indexer --root /path/to/workspace
```

Or target a single file:

```bash
cd indexer
PYTHONPATH=src python3 -m cortado_indexer --file /path/to/file.dart
```

## Notes

- Dart currently stays on the fallback chunker path in this scaffold until a verified parser binding/build flow is wired in.
- The output is newline-delimited JSON to keep the first job integration simple.
