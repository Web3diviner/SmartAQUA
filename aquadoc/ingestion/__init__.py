"""Knowledge ingestion pipeline.

    load -> parse -> clean -> chunk -> tag metadata -> embed -> store (pending)

Kept as a separate top-level package from `app` because ingestion is an offline
batch concern with a different failure model from request serving: a partial
ingest is recoverable and reviewable, a partial chat response is not.
"""

from ingestion.chunker import Chunk, Chunker, estimate_tokens
from ingestion.cleaner import clean_text
from ingestion.loader import ALLOWED_TYPES, LoadedDocument, compute_checksum, load_bytes, load_path
from ingestion.parser import ParsedBlock, ParsedDocument, parse
from ingestion.service import IngestionConfig, IngestionService

__all__ = [
    "ALLOWED_TYPES",
    "Chunk",
    "Chunker",
    "IngestionConfig",
    "IngestionService",
    "LoadedDocument",
    "ParsedBlock",
    "ParsedDocument",
    "clean_text",
    "compute_checksum",
    "estimate_tokens",
    "load_bytes",
    "load_path",
    "parse",
]
