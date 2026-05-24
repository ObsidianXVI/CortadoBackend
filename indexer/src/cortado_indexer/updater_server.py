from __future__ import annotations

import json
import logging
import os
import threading
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from .updater import (
    FILE_CHANGE_WINDOW_SECONDS,
    IncrementalIndexProcessor,
    LocalWorkspaceFileLoader,
    NoopBatchProcessor,
    WorkspaceEventBatcher,
    default_qdrant_client_factory,
    decode_ingest_request,
    run_batch_worker,
)
from .embedding import VertexAIEmbedder

logging.basicConfig(level=os.getenv("CORTADO_LOG_LEVEL", "INFO"))
logger = logging.getLogger(__name__)


class UpdaterApplication:
    def __init__(
        self,
        *,
        batcher: WorkspaceEventBatcher | None = None,
        processor: Any | None = None,
    ) -> None:
        self.batcher = batcher or WorkspaceEventBatcher()
        self.processor = processor or NoopBatchProcessor()

    @classmethod
    def from_env(cls) -> "UpdaterApplication":
        window_seconds = float(
            os.getenv(
                "CORTADO_UPDATER_BATCH_WINDOW_SECONDS",
                str(FILE_CHANGE_WINDOW_SECONDS),
            )
        )
        batcher = WorkspaceEventBatcher(window_seconds=window_seconds)

        mode = os.getenv("CORTADO_UPDATER_MODE", "noop").strip().lower()
        if mode == "local":
            processor = IncrementalIndexProcessor(
                embedder=VertexAIEmbedder.from_env(),
                file_loader=LocalWorkspaceFileLoader.from_env(),
                qdrant_client_factory=default_qdrant_client_factory,
            )
        else:
            processor = NoopBatchProcessor()

        return cls(batcher=batcher, processor=processor)

    def ingest(self, payload: dict[str, object]) -> int:
        events = decode_ingest_request(payload)
        for event in events:
            self.batcher.add(event)
        return len(events)


def main() -> int:
    app = UpdaterApplication.from_env()
    stop_event = threading.Event()
    worker = threading.Thread(
        target=run_batch_worker,
        kwargs={
            "batcher": app.batcher,
            "processor": app.processor,
            "stop_event": stop_event,
        },
        daemon=True,
    )
    worker.start()

    port = int(os.getenv("PORT", "8080"))
    server = ThreadingHTTPServer(("0.0.0.0", port), _handler_for_application(app))

    logger.info("serving cortado indexer updater on 0.0.0.0:%d", port)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("received shutdown signal")
    finally:
        server.shutdown()
        stop_event.set()
        worker.join(timeout=5)
    return 0


def _handler_for_application(app: UpdaterApplication) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            if self.path != "/healthz":
                self._write_json(
                    HTTPStatus.NOT_FOUND,
                    {"error": "not found"},
                )
                return
            self._write_json(HTTPStatus.OK, {"status": "ok"})

        def do_POST(self) -> None:
            if self.path != "/ingest":
                self._write_json(
                    HTTPStatus.NOT_FOUND,
                    {"error": "not found"},
                )
                return

            try:
                payload = self._read_json()
                ingested = app.ingest(payload)
            except Exception as err:
                self._write_json(
                    HTTPStatus.BAD_REQUEST,
                    {"error": str(err)},
                )
                return

            self._write_json(
                HTTPStatus.ACCEPTED,
                {"ingested": ingested, "status": "queued"},
            )

        def log_message(self, format: str, *args: object) -> None:
            logger.info("%s - %s", self.address_string(), format % args)

        def _read_json(self) -> dict[str, object]:
            content_length = int(self.headers.get("Content-Length", "0"))
            body = self.rfile.read(content_length)
            if not body:
                raise ValueError("request body is required")
            payload = json.loads(body.decode("utf-8"))
            if not isinstance(payload, dict):
                raise ValueError("request body must be a JSON object")
            return payload

        def _write_json(self, status: HTTPStatus, payload: dict[str, object]) -> None:
            body = json.dumps(payload, sort_keys=True).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return Handler


if __name__ == "__main__":
    raise SystemExit(main())
