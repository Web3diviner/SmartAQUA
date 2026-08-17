"""Embedding access, isolated behind `EmbeddingProvider`."""

from app.config import Settings
from app.embeddings.base import EmbeddingProvider
from app.embeddings.hashing import HashingEmbeddingProvider


def build_embedding_provider(settings: Settings) -> EmbeddingProvider:
    if settings.embedding_provider == "voyage":
        from app.embeddings.voyage import VoyageEmbeddingProvider

        return VoyageEmbeddingProvider(
            api_key=settings.voyage_api_key,
            model=settings.voyage_model,
            dimensions=settings.embedding_dimensions,
            timeout_seconds=settings.embedding_timeout_seconds,
        )

    if settings.embedding_provider == "hashing":
        return HashingEmbeddingProvider(dimensions=settings.embedding_dimensions)

    raise ValueError(f"Unsupported EMBEDDING_PROVIDER: {settings.embedding_provider}")


__all__ = ["EmbeddingProvider", "HashingEmbeddingProvider", "build_embedding_provider"]
