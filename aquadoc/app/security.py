"""Authentication for the two caller classes AquaDoc accepts.

05_API_AND_SERVICE_CONTRACTS.md: AquaDoc is an internal service. The Go backend
owns user authentication and authorization; AquaDoc only verifies that the
caller is the Go backend (service credential) or, in development, a developer
using the temporary web frontend.

AquaDoc must never become a second public authentication system
(15_AQUADOC_FRONTEND.md section 8).
"""

from __future__ import annotations

import hmac
from dataclasses import dataclass
from typing import Literal

from fastapi import Request

from app.config import Settings, get_settings
from app.errors import ForbiddenError, UnauthorizedError

CallerKind = Literal["service", "developer"]


@dataclass(frozen=True)
class Caller:
    """Who is making this request, as far as AquaDoc can tell."""

    kind: CallerKind
    # For service calls this is the user the Go backend is acting on behalf of.
    # AquaDoc trusts it for attribution only — never for authorization.
    subject: str

    @property
    def is_developer(self) -> bool:
        """Whether this caller may see prompt internals and retrieval traces.

        Only true for the development token. A service call stays false even
        when the Go backend is itself running in development, so debug payloads
        never reach a farmer-facing client by accident.
        """
        return self.kind == "developer"


def _bearer_token(request: Request) -> str:
    header = request.headers.get("authorization", "")
    scheme, _, token = header.partition(" ")
    if scheme.lower() != "bearer" or not token.strip():
        raise UnauthorizedError("Missing or malformed Authorization header.")
    return token.strip()


def _constant_time_equals(candidate: str, expected: str) -> bool:
    if not expected:
        return False
    return hmac.compare_digest(candidate.encode("utf-8"), expected.encode("utf-8"))


def _settings_for(request: Request) -> Settings:
    """Read settings from app state, falling back to the process defaults.

    Using app state rather than `Depends(get_settings)` means a test (or a
    second app instance) that injects its own Settings is authenticated against
    those settings, not against whatever the environment happens to hold.
    """
    return getattr(request.app.state, "settings", None) or get_settings()


def require_service_caller(request: Request) -> Caller:
    """Authenticate the Go backend on /internal/v1/*.

    In production this should sit behind private networking or mTLS as well;
    the shared secret is defence in depth, not the only boundary
    (07_SECURITY_ARCHITECTURE.md section 2).
    """
    settings = _settings_for(request)
    if not settings.aquadoc_internal_service_secret:
        # Refuse rather than silently allowing unauthenticated internal calls.
        raise ForbiddenError("Internal API is not configured on this instance.")

    token = _bearer_token(request)

    # A developer token is also accepted here so the temporary web client can
    # exercise the real internal contract. It yields a developer Caller, so the
    # debug payloads follow the caller, not the route.
    if settings.dev_routes_enabled and _constant_time_equals(token, settings.aquadoc_dev_token):
        return Caller(kind="developer", subject=_subject(request, "developer"))

    if not _constant_time_equals(token, settings.aquadoc_internal_service_secret):
        raise UnauthorizedError("Invalid service credential.")

    return Caller(kind="service", subject=_subject(request, "unknown"))


def require_dev_caller(request: Request) -> Caller:
    """Authenticate a developer on /dev/v1/*.

    Only reachable when APP_ENV=development and AQUADOC_DEV_TOKEN is set. The
    settings validator refuses to boot a production instance with a dev token,
    and main.py does not mount these routes outside development — so this path
    cannot exist in production.
    """
    settings = _settings_for(request)
    if not settings.dev_routes_enabled:
        raise ForbiddenError("Development API is disabled on this instance.")

    token = _bearer_token(request)
    if not _constant_time_equals(token, settings.aquadoc_dev_token):
        raise UnauthorizedError("Invalid development token.")

    return Caller(kind="developer", subject=_subject(request, "developer"))


def _subject(request: Request, default: str) -> str:
    """Who the caller is acting for, for attribution only.

    Never used for an authorization decision: the Go backend owns farm
    ownership and pond permission checks (15_AQUADOC_FRONTEND.md section 8).
    """
    return request.headers.get("x-aquadoc-user-id", "").strip()[:128] or default
