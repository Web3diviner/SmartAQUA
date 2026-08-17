"""Voyage AI implementation of `EmbeddingProvider`.

A hosted embeddings API, used as the production default. Swapping to another
hosted provider or a local model means adding a sibling module and changing
`EMBEDDING_PROVIDER` — no call site changes.
"""

from __future__ import annotations

import logging

import httpx

from app.embeddings.base import EmbeddingProvider
from app.errors import EmbeddingProviderError

logger = logging.getLogger(__name__)

_API_URL = "https://api.voyageai.com/v1/embeddings"
#: The API rejects oversized batches; chunking here keeps ingestion resilient.
_MAX_BATCH = 96


class VoyageEmbeddingProvider(EmbeddingProvider):
    name = "voyage"

    def __init__(
        self,
        *,
        api_key: str,
        model: str = "voyage-3",
        dimensions: int = 1024,
        timeout_seconds: float = 60.0,
    ) -> None:
        if not api_key:
            raise EmbeddingProviderError("VOYAGE_API_KEY is not configured.")
        self._model = model
        self._dimensions = dimensions
        self._client = httpx.AsyncClient(
            timeout=httpx.Timeout(timeout_seconds),
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
        )

    @property
    def model_id(self) -> str:
        return self._model

    @property
    def dimensions(self) -> int:
        return self._dimensions

    async def embed_documents(self, texts: list[str]) -> list[list[float]]:
        if not texts:
            return []
        vectors: list[list[float]] = []
        for start in range(0, len(texts), _MAX_BATCH):
            batch = texts[start : start + _MAX_BATCH]
            vectors.extend(await self._request(batch, input_type="document"))
        return vectors

    async def embed_query(self, text: str) -> list[float]:
        vectors = await self._request([text], input_type="query")
        return vectors[0]

    async def _request(self, texts: list[str], *, input_type: str) -> list[list[float]]:
        payload = {
            "model": self._model,
            "input": texts,
            "input_type": input_type,
            "output_dimension": self._dimensions,
        }
        try:
            response = await self._client.post(_API_URL, json=payload)
            response.raise_for_status()
            body = response.json()
        except httpx.HTTPError as exc:
            logger.exception("voyage_request_failed", extra={"model": self._model})
            raise EmbeddingProviderError("The embedding provider request failed.") from exc

        try:
            # The API does not guarantee response order; sort by index.
            items = sorted(body["data"], key=lambda item: item["index"])
            vectors = [item["embedding"] for item in items]
        except (KeyError, TypeError) as exc:
            raise EmbeddingProviderError("The embedding provider returned an unexpected shape.") from exc

        if len(vectors) != len(texts):
            raise EmbeddingProviderError(
                f"Embedding count mismatch: requested {len(texts)}, received {len(vectors)}."
            )
        for vector in vectors:
            if len(vector) != self._dimensions:
                raise EmbeddingProviderError(
                    f"Embedding dimension mismatch: expected {self._dimensions}, "
                    f"received {len(vector)}. The database vector column and "
                    f"EMBEDDING_DIMENSIONS must agree with the model."
                )
        return vectors

    async def aclose(self) -> None:
        await self._client.aclose()
