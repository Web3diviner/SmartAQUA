"""Deterministic offline embedding provider.

Hashed character n-grams projected into a fixed-dimension vector, L2-normalised.
It captures lexical overlap only — no semantics, no synonymy. It exists so the
pipeline runs end to end without an API key and so retrieval tests are
deterministic.

The settings validator refuses to start a production instance configured with
this provider; production must use a real embedding model.
"""

from __future__ import annotations

import hashlib
import math
import re
from collections import Counter

from app.embeddings.base import EmbeddingProvider

_TOKEN_RE = re.compile(r"[a-z0-9]+")
_NGRAM_SIZES = (3, 4, 5)


class HashingEmbeddingProvider(EmbeddingProvider):
    name = "hashing"

    def __init__(self, dimensions: int = 1024) -> None:
        if dimensions <= 0:
            raise ValueError("dimensions must be positive")
        self._dimensions = dimensions

    @property
    def model_id(self) -> str:
        return f"hashing-ngram-v1-{self._dimensions}"

    @property
    def dimensions(self) -> int:
        return self._dimensions

    async def embed_documents(self, texts: list[str]) -> list[list[float]]:
        return [self._embed(text) for text in texts]

    async def embed_query(self, text: str) -> list[float]:
        return self._embed(text)

    def _embed(self, text: str) -> list[float]:
        counts: Counter[str] = Counter()
        tokens = _TOKEN_RE.findall(text.lower())

        for token in tokens:
            counts[f"w:{token}"] += 1
            padded = f"^{token}$"
            for size in _NGRAM_SIZES:
                for start in range(max(0, len(padded) - size + 1)):
                    counts[f"g:{padded[start : start + size]}"] += 1

        vector = [0.0] * self._dimensions
        for feature, count in counts.items():
            index, sign = self._bucket(feature)
            # Sub-linear term weighting keeps a repeated word from dominating.
            vector[index] += sign * (1.0 + math.log(count))

        norm = math.sqrt(sum(value * value for value in vector))
        if norm == 0.0:
            return vector
        return [value / norm for value in vector]

    def _bucket(self, feature: str) -> tuple[int, float]:
        digest = hashlib.blake2b(feature.encode("utf-8"), digest_size=8).digest()
        raw = int.from_bytes(digest, "big")
        # Signed hashing: halves systematic collision bias.
        return raw % self._dimensions, 1.0 if (raw >> 63) & 1 else -1.0
