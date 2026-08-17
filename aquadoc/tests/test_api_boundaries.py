"""API boundaries: authentication, error envelope, and surface separation.

These exercise the real ASGI app. Retrieval needs pgvector, so the chat paths
are covered by their unit tests instead; what matters here is that the security
boundary and the error contract hold before any database work begins.
"""

from __future__ import annotations

from collections.abc import Iterator

import pytest
from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app

DEV_TOKEN = "test-dev-token"
SERVICE_SECRET = "test-service-secret-that-is-long-enough"


def _client(settings: Settings) -> Iterator[TestClient]:
    """Yield a client with the lifespan run, so app.state is populated.

    `raise_server_exceptions=False` so handler behaviour is asserted through the
    response, exactly as a real caller would see it.
    """
    app = create_app(settings)
    with TestClient(app, raise_server_exceptions=False) as client:
        yield client


@pytest.fixture
def dev_client() -> Iterator[TestClient]:
    yield from _client(
        Settings(
            app_env="development",
            aquadoc_dev_token=DEV_TOKEN,
            aquadoc_internal_service_secret=SERVICE_SECRET,
            llm_provider="echo",
            embedding_provider="hashing",
        )
    )


@pytest.fixture
def staging_client() -> Iterator[TestClient]:
    """Non-development, non-production: dev routes must not be mounted."""
    yield from _client(
        Settings(
            app_env="staging",
            aquadoc_dev_token="",
            aquadoc_internal_service_secret=SERVICE_SECRET,
            llm_provider="echo",
            embedding_provider="hashing",
        )
    )


def _chat_payload() -> dict:
    return {"user_id": "USER-1", "question": "What is FCR?"}


class TestAuthenticationBoundary:
    def test_internal_chat_requires_a_credential(self, dev_client: TestClient) -> None:
        response = dev_client.post("/internal/v1/aquadoc/chat", json=_chat_payload())

        assert response.status_code == 401
        assert response.json()["error"]["code"] == "UNAUTHORIZED"

    def test_wrong_credential_is_rejected(self, dev_client: TestClient) -> None:
        response = dev_client.post(
            "/internal/v1/aquadoc/chat",
            json=_chat_payload(),
            headers={"Authorization": "Bearer wrong-secret"},
        )

        assert response.status_code == 401

    def test_malformed_authorization_header_is_rejected(self, dev_client: TestClient) -> None:
        for header in ("", "Basic abc", "Bearer", "Bearer   "):
            response = dev_client.post(
                "/internal/v1/aquadoc/chat",
                json=_chat_payload(),
                headers={"Authorization": header},
            )
            assert response.status_code == 401, header

    def test_dev_routes_reject_the_service_secret(self, dev_client: TestClient) -> None:
        """The developer surface is not reachable with a service credential."""
        response = dev_client.get(
            "/dev/v1/knowledge/documents",
            headers={"Authorization": f"Bearer {SERVICE_SECRET}"},
        )

        assert response.status_code == 401

    def test_dev_config_route_accepts_the_dev_token(self, dev_client: TestClient) -> None:
        response = dev_client.get(
            "/dev/v1/config", headers={"Authorization": f"Bearer {DEV_TOKEN}"}
        )

        assert response.status_code == 200
        body = response.json()
        assert body["environment"] == "development"
        assert body["llm_provider"] == "echo"

    def test_config_never_exposes_secrets(self, dev_client: TestClient) -> None:
        """07_SECURITY_ARCHITECTURE.md section 4: no secrets in responses."""
        response = dev_client.get(
            "/dev/v1/config", headers={"Authorization": f"Bearer {DEV_TOKEN}"}
        )
        serialised = response.text.lower()

        assert DEV_TOKEN.lower() not in serialised
        assert SERVICE_SECRET.lower() not in serialised
        for forbidden in ("api_key", "password", "secret", "database_url"):
            assert forbidden not in serialised


class TestSurfaceSeparation:
    def test_dev_routes_absent_outside_development(self, staging_client: TestClient) -> None:
        """Structural, not a per-route flag check."""
        response = staging_client.get(
            "/dev/v1/config", headers={"Authorization": f"Bearer {DEV_TOKEN}"}
        )

        assert response.status_code == 404

    def test_internal_routes_present_outside_development(self, staging_client: TestClient) -> None:
        response = staging_client.post("/internal/v1/aquadoc/chat", json=_chat_payload())

        # Reachable, and correctly refusing an unauthenticated caller.
        assert response.status_code == 401


class TestErrorEnvelope:
    def test_shape_matches_the_service_contract(self, dev_client: TestClient) -> None:
        """05_API_AND_SERVICE_CONTRACTS.md, "Error Format"."""
        error = dev_client.post("/internal/v1/aquadoc/chat", json=_chat_payload()).json()["error"]

        assert set(error) >= {"code", "message", "request_id"}
        assert error["request_id"]

    def test_validation_errors_do_not_echo_submitted_values(
        self, dev_client: TestClient
    ) -> None:
        """Farmer-supplied text must not be reflected back in an error body."""
        response = dev_client.post(
            "/dev/v1/chat",
            json={"user_id": "USER-1", "question": ""},
            headers={"Authorization": f"Bearer {DEV_TOKEN}"},
        )

        assert response.status_code == 422
        body = response.json()
        assert body["error"]["code"] == "VALIDATION_ERROR"
        assert "input" not in response.text

    def test_unknown_route_uses_the_same_envelope(self, dev_client: TestClient) -> None:
        response = dev_client.get("/no/such/route")

        assert response.status_code == 404
        assert response.json()["error"]["code"] == "NOT_FOUND"

    def test_request_id_is_echoed_for_tracing(self, dev_client: TestClient) -> None:
        response = dev_client.post(
            "/internal/v1/aquadoc/chat",
            json=_chat_payload(),
            headers={"X-Request-ID": "REQ-FROM-GO-BACKEND"},
        )

        assert response.headers["X-Request-ID"] == "REQ-FROM-GO-BACKEND"
        assert response.json()["error"]["request_id"] == "REQ-FROM-GO-BACKEND"

    def test_request_id_generated_when_absent(self, dev_client: TestClient) -> None:
        response = dev_client.post("/internal/v1/aquadoc/chat", json=_chat_payload())

        assert response.headers["X-Request-ID"].startswith("REQ-")


class TestHealth:
    def test_liveness_needs_no_credential(self, dev_client: TestClient) -> None:
        response = dev_client.get("/health")

        assert response.status_code == 200
        assert response.json()["status"] == "ok"

    def test_liveness_reveals_no_configuration(self, dev_client: TestClient) -> None:
        body = dev_client.get("/health").text.lower()

        assert "postgres" not in body
        assert "password" not in body
