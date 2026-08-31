"""Tests for multi-turn conversation memory and conversation history endpoints."""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app
from app.orchestration.context_builder import build_conversation_history_block, build_user_turn

DEV_TOKEN = "test-dev-token"
SERVICE_SECRET = "test-service-secret-that-is-long-enough"


@pytest.fixture
def dev_client():
    app = create_app(
        Settings(
            app_env="development",
            aquadoc_dev_token=DEV_TOKEN,
            aquadoc_internal_service_secret=SERVICE_SECRET,
            llm_provider="echo",
            embedding_provider="hashing",
        )
    )
    with TestClient(app, raise_server_exceptions=False) as client:
        yield client


def test_build_conversation_history_block_empty():
    assert build_conversation_history_block(None) == ""
    assert build_conversation_history_block([]) == ""


def test_build_conversation_history_block_multi_turn():
    history = [
        {"role": "user", "content": "My catfish have fin rot."},
        {"role": "assistant", "content": "Fin rot can be caused by Aeromonas bacteria."},
        {"role": "user", "content": "What dosage of bitter leaf extract should I use?"},
        {"role": "assistant", "content": "Apply 10-15g per 100 liters of water."},
    ]
    block = build_conversation_history_block(history)
    assert "<conversation_history>" in block
    assert "<farmer>" in block
    assert "My catfish have fin rot." in block
    assert "<doctor>" in block
    assert "Fin rot can be caused by Aeromonas bacteria." in block
    assert "What dosage of bitter leaf extract should I use?" in block
    assert "</conversation_history>" in block


def test_build_conversation_history_block_retains_up_to_10_turns():
    # 12 turns (24 messages)
    history = []
    for i in range(12):
        history.append({"role": "user", "content": f"Turn-{i+1:02d}-Question"})
        history.append({"role": "assistant", "content": f"Turn-{i+1:02d}-Answer"})

    block = build_conversation_history_block(history)
    # The first 2 turns (Turn 01 and Turn 02) should be dropped; last 10 turns (Turn 03 to 12) retained
    assert "Turn-01-Question" not in block
    assert "Turn-02-Question" not in block
    assert "Turn-03-Question" in block
    assert "Turn-12-Question" in block


def test_build_user_turn_includes_history():
    history = [
        {"role": "user", "content": "Pond 1 has low DO."},
        {"role": "assistant", "content": "Aerate immediately and reduce feeding."},
    ]
    turn_str = build_user_turn(
        question="Can I feed now?",
        context=None,
        findings=[],
        candidates=[],
        history=history,
    )
    assert "<conversation_history>" in turn_str
    assert "Pond 1 has low DO." in turn_str
    assert "<question>\nCan I feed now?\n</question>" in turn_str


def test_dev_conversations_api_crud(dev_client: TestClient):
    headers = {"Authorization": f"Bearer {DEV_TOKEN}"}

    # 1. Send first chat message to initiate a conversation
    resp1 = dev_client.post(
        "/dev/v1/chat",
        headers=headers,
        json={
            "user_id": "farmer-1",
            "question": "My fish have white spots on their skin.",
            "conversation_id": "test-conv-001",
        },
    )
    assert resp1.status_code == 200
    data1 = resp1.json()
    assert "answer" in data1

    # 2. List conversations
    list_resp = dev_client.get("/dev/v1/conversations", headers=headers)
    assert list_resp.status_code == 200
    list_data = list_resp.json()
    assert "conversations" in list_data
    conv_ids = [c["id"] for c in list_data["conversations"]]
    assert "test-conv-001" in conv_ids

    # 3. Get single conversation detail
    detail_resp = dev_client.get("/dev/v1/conversations/test-conv-001", headers=headers)
    assert detail_resp.status_code == 200
    detail_data = detail_resp.json()
    assert detail_data["id"] == "test-conv-001"
    assert len(detail_data.get("messages", [])) >= 2

    # 4. Delete conversation
    del_resp = dev_client.delete("/dev/v1/conversations/test-conv-001", headers=headers)
    assert del_resp.status_code == 200
    assert del_resp.json()["success"] is True

    # 5. Verify deleted
    detail_resp_2 = dev_client.get("/dev/v1/conversations/test-conv-001", headers=headers)
    assert detail_resp_2.status_code == 200
    assert "error" in detail_resp_2.json() or detail_resp_2.json().get("status_code") == 404
