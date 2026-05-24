# Cortado Indexer

`indexer/` is the Python job scaffold for v0.5 Feature 5.1.

Current scope:
- chunk source files for indexing
- use semantic chunking for Python, JavaScript, and Go
- fall back to overlapping line windows when no parser is available yet

The initial scaffold keeps the dependency surface conservative:
- `tree-sitter==0.22.0` is pinned per the release spec
- grammar wheels are pinned only for the first verified semantic language set
- language-specific parser wiring stays isolated behind a registry
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

- Python, JavaScript, and Go now use parser-backed semantic chunks when their grammar packages are installed.
- Dart currently stays on the fallback chunker path until a verified parser binding/build flow is added.
- The output is newline-delimited JSON to keep the first job integration simple.
