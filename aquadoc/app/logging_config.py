"""Structured JSON logging with request correlation.

07_SECURITY_ARCHITECTURE.md section 10: never log passwords, tokens, API keys,
or secrets; use request IDs. The formatter below emits a fixed field set and
scrubs any value that looks like a credential from the message.
"""

from __future__ import annotations

import json
import logging
import re
import sys
from contextvars import ContextVar
from typing import Any

request_id_var: ContextVar[str] = ContextVar("request_id", default="-")

_SECRET_PATTERNS = (
    re.compile(r"(sk-[A-Za-z0-9_\-]{8,})"),
    re.compile(r"(Bearer\s+[A-Za-z0-9._\-]{8,})", re.IGNORECASE),
    re.compile(r"((?:api[_-]?key|token|secret|password)\"?\s*[:=]\s*\"?)([^\s\",}]{4,})", re.I),
)

_RESERVED = frozenset(logging.LogRecord("", 0, "", 0, "", (), None).__dict__) | {
    "asctime",
    "message",
    "taskName",
}


def scrub(text: str) -> str:
    """Redact anything that pattern-matches a credential."""
    scrubbed = _SECRET_PATTERNS[0].sub("[redacted]", text)
    scrubbed = _SECRET_PATTERNS[1].sub("Bearer [redacted]", scrubbed)
    return _SECRET_PATTERNS[2].sub(r"\1[redacted]", scrubbed)


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "timestamp": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"),
            "level": record.levelname,
            "logger": record.name,
            "message": scrub(record.getMessage()),
            "request_id": request_id_var.get(),
        }
        for key, value in record.__dict__.items():
            if key not in _RESERVED and not key.startswith("_"):
                payload[key] = value
        if record.exc_info:
            # Exception type and message only — stack traces stay out of the log
            # payload and never reach an API response.
            exc_type, exc_value, _ = record.exc_info
            payload["error_type"] = getattr(exc_type, "__name__", str(exc_type))
            payload["error_message"] = scrub(str(exc_value))
        return json.dumps(payload, default=str)


def configure_logging(level: str = "INFO", *, json_logs: bool = True) -> None:
    """Install the root handler.

    JSON is the production format (machine-parseable, scrubbed). Local
    development defaults to a plain format that is readable in a terminal —
    scrubbing still applies to both.
    """
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(JsonFormatter() if json_logs else _PlainFormatter())

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(level.upper())

    # uvicorn installs its own handlers; route them through ours.
    for name in ("uvicorn", "uvicorn.access", "uvicorn.error"):
        uvicorn_logger = logging.getLogger(name)
        uvicorn_logger.handlers.clear()
        uvicorn_logger.propagate = True


class _PlainFormatter(logging.Formatter):
    """Human-readable local format. Applies the same secret scrubbing."""

    def format(self, record: logging.LogRecord) -> str:
        base = (
            f"{self.formatTime(record, '%H:%M:%S')} {record.levelname:<7} "
            f"[{request_id_var.get()}] {record.name}: {scrub(record.getMessage())}"
        )
        extras = {
            key: value
            for key, value in record.__dict__.items()
            if key not in _RESERVED and not key.startswith("_")
        }
        if extras:
            base += " " + " ".join(f"{key}={value}" for key, value in extras.items())
        if record.exc_info:
            exc_type, exc_value, _ = record.exc_info
            base += (
                f" error={getattr(exc_type, '__name__', exc_type)}: {scrub(str(exc_value))}"
            )
        return base
