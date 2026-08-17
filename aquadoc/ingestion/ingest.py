"""CLI for ingesting knowledge documents.

    python -m ingestion.ingest --file guide.pdf \
        --title "FAO Aquaculture Manual" \
        --source "FAO" \
        --document-type guideline \
        --evidence-level A \
        --species "Clarias gariepinus" \
        --topic feeding --topic water_quality

Documents land in `pending`. Approve them from the Knowledge Base screen or via
POST /dev/v1/knowledge/documents/{id}/approve — ingesting is not approving.
"""

from __future__ import annotations

import argparse
import asyncio
import sys
from pathlib import Path

from app.config import get_settings
from app.db import Database
from app.embeddings import build_embedding_provider
from app.errors import AquaDocError
from app.logging_config import configure_logging
from app.schemas.common import EvidenceLevel
from app.schemas.knowledge import DocumentMetadata
from ingestion.loader import load_path
from ingestion.service import IngestionConfig, IngestionService


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aquadoc-ingest",
        description="Ingest an aquaculture knowledge document into AquaDoc.",
    )
    parser.add_argument("--file", required=True, type=Path, help="Path to a PDF, TXT, or MD file.")
    parser.add_argument("--title", required=True, help="Document title.")
    parser.add_argument("--source", required=True, help="Publisher or origin, e.g. 'FAO'.")
    parser.add_argument("--author", default=None)
    parser.add_argument("--year", type=int, default=None)
    parser.add_argument(
        "--document-type",
        required=True,
        help="guideline | research_paper | manual | expert_case | user_report | other",
    )
    parser.add_argument(
        "--evidence-level",
        required=True,
        choices=[level.value for level in EvidenceLevel],
        help="A=official guideline, B=peer-reviewed, C=textbook, D=expert case, E=user report.",
    )
    parser.add_argument("--species", action="append", default=[])
    parser.add_argument("--life-stage", action="append", default=[])
    parser.add_argument("--topic", action="append", default=[])
    parser.add_argument("--disease", action="append", default=[])
    parser.add_argument("--region", action="append", default=[])
    parser.add_argument("--owner", default=None, help="Who is accountable for this document.")
    parser.add_argument(
        "--replace-existing",
        action="store_true",
        help="Re-ingest as a new version when the checksum already exists.",
    )
    return parser


async def run(args: argparse.Namespace) -> int:
    settings = get_settings()
    configure_logging(settings.log_level)

    metadata = DocumentMetadata(
        title=args.title,
        source=args.source,
        author=args.author,
        year=args.year,
        document_type=args.document_type,
        species=args.species,
        life_stage=args.life_stage,
        topic=args.topic,
        disease=args.disease,
        region=args.region,
        evidence_level=EvidenceLevel(args.evidence_level),
        owner=args.owner,
    )

    database = Database(settings)
    embeddings = build_embedding_provider(settings)
    service = IngestionService(
        embeddings=embeddings,
        config=IngestionConfig(
            chunk_target_tokens=settings.chunk_target_tokens,
            chunk_overlap_tokens=settings.chunk_overlap_tokens,
        ),
    )

    try:
        document = load_path(args.file, max_bytes=settings.max_upload_bytes)
        async with database.session() as session:
            result = await service.ingest(
                session,
                document=document,
                metadata=metadata,
                replace_existing=args.replace_existing,
            )
    except AquaDocError as exc:
        print(f"Ingestion failed [{exc.code}]: {exc.message}", file=sys.stderr)
        return 1
    finally:
        await embeddings.aclose()
        await database.dispose()

    print(f"Ingested '{result.title}'")
    print(f"  document_id   : {result.document_id}")
    print(f"  checksum      : {result.checksum}")
    print(f"  chunks        : {result.chunk_count} ({result.embedded_chunks} embedded)")
    print(f"  review status : {result.review_status.value}")
    for warning in result.warnings:
        print(f"  warning       : {warning}")
    print()
    print("This document is NOT yet retrievable. Approve it before it enters production RAG.")
    return 0


def main() -> None:
    args = build_parser().parse_args()
    raise SystemExit(asyncio.run(run(args)))


if __name__ == "__main__":
    main()
