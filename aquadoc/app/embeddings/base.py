"""Embedding provider abstraction.

04_AQUADOC_RAG_LLM.md section 11. Embeddings are model-specific: vectors
produced by one model are not comparable with another's. Changing
`EMBEDDING_PROVIDER` or the model requires a full re-ingest, which is why
`model_id` is recorded on every chunk and in provenance.
"""

from __future__ import annotations

from abc import ABC, abstractmethod


class EmbeddingProvider(ABC):
    name: str = "abstract"

    @property
    @abstractmethod
    def model_id(self) -> str:
        raise NotImplementedError

    @property
    @abstractmethod
    def dimensions(self) -> int:
        raise NotImplementedError

    @abstractmethod
    async def embed_documents(self, texts: list[str]) -> list[list[float]]:
        """Embed passages for storage. Order matches the input."""
        raise NotImplementedError

    @abstractmethod
    async def embed_query(self, text: str) -> list[float]:
        """Embed a single search query.

        Kept separate from `embed_documents` because several providers use an
        asymmetric query/document encoding.
        """
        raise NotImplementedError

    async def aclose(self) -> None:
        return None
