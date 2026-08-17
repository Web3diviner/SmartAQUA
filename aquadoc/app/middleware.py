"""Request-scoped middleware: correlation IDs and access logging.

05_API_AND_SERVICE_CONTRACTS.md requires `request_id` in every error body, and
07_SECURITY_ARCHITECTURE.md section 10 requires request IDs in logs. Both need
the ID to exist before any handler runs, which is why it is assigned here rather
than inside a route.
"""

from __future__ import annotations

import logging
import time
import uuid

from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response

from app.logging_config import request_id_var

logger = logging.getLogger(__name__)

REQUEST_ID_HEADER = "X-Request-ID"


class RequestContextMiddleware(BaseHTTPMiddleware):
    """Assign a request ID, bind it to logging, and echo it back."""

    async def dispatch(self, request: Request, call_next: RequestResponseEndpoint) -> Response:
        # Reuse the caller's ID so a single farmer question can be traced across
        # Flutter -> Go backend -> AquaDoc.
        incoming = request.headers.get(REQUEST_ID_HEADER, "").strip()
        request_id = incoming[:128] if incoming else f"REQ-{uuid.uuid4().hex[:12].upper()}"

        token = request_id_var.set(request_id)
        request.state.request_id = request_id
        started = time.perf_counter()

        try:
            response = await call_next(request)
        except Exception:
            # Logged here so the duration is captured even on an unhandled
            # error; the exception handlers own the response body.
            logger.exception(
                "request_failed",
                extra={
                    "method": request.method,
                    "path": request.url.path,
                    "duration_ms": round((time.perf_counter() - started) * 1000, 2),
                },
            )
            raise
        finally:
            request_id_var.reset(token)

        duration_ms = round((time.perf_counter() - started) * 1000, 2)
        response.headers[REQUEST_ID_HEADER] = request_id

        # Query strings are omitted: they can carry farmer-supplied text.
        logger.info(
            "request_completed",
            extra={
                "method": request.method,
                "path": request.url.path,
                "status_code": response.status_code,
                "duration_ms": duration_ms,
            },
        )
        return response
