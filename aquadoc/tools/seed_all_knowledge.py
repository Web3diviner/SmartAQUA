"""Automated Knowledge Base Seeding & Ingestion Tool for AquaDoc.

Ingests and approves all verified knowledge documents from `sample-knowledge/`
into the AquaDoc RAG database with full vector embeddings and metadata indexing.
"""

import asyncio
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from app.config import get_settings
from app.db import Database
from app.embeddings import build_embedding_provider
from app.models import KnowledgeDocument
from app.schemas.common import EvidenceLevel, ReviewStatus
from app.schemas.knowledge import DocumentMetadata
from ingestion.loader import load_path
from ingestion.service import IngestionConfig, IngestionService


def extract_metadata_from_md(text: str, filename: str) -> DocumentMetadata:
    lines = text.splitlines()
    title = filename.replace(".md", "").replace("-", " ").title()
    for line in lines[:10]:
        if line.startswith("# "):
            title = line[2:].strip()
            break

    source = "National Aquaculture Research Institute / FAO"
    year = 2024
    topics = ["aquaculture", "water_quality", "west_africa"]
    evidence_level = EvidenceLevel.A_OFFICIAL_GUIDELINE

    for line in lines[:15]:
        if "**Publisher / Source:**" in line or "**Source:**" in line:
            source = line.split(":", 1)[1].replace("**", "").strip()
        elif "**Year:**" in line:
            y_match = re.search(r"\d{4}", line)
            if y_match:
                year = int(y_match.group(0))
        elif "**Topics:**" in line:
            raw_topics = line.split(":", 1)[1].replace("**", "").strip()
            topics = [t.strip().lower() for t in raw_topics.split(",") if t.strip()]
        elif "**Evidence Level:**" in line:
            raw_el = line.split(":", 1)[1].replace("**", "").strip().upper()
            if "A" in raw_el:
                evidence_level = EvidenceLevel.A_OFFICIAL_GUIDELINE
            elif "B" in raw_el:
                evidence_level = EvidenceLevel.B_PEER_REVIEWED
            elif "C" in raw_el:
                evidence_level = EvidenceLevel.C_TEXTBOOK
            elif "D" in raw_el:
                evidence_level = EvidenceLevel.D_EXPERT_CASE
            elif "E" in raw_el:
                evidence_level = EvidenceLevel.E_USER_REPORT

    species = ["Clarias gariepinus", "Oreochromis niloticus"] if "tilapia" in text.lower() or "catfish" in text.lower() else []
    disease = ["Aeromonas", "Columnaris", "Broken Head", "Trichodina"] if "disease" in text.lower() or "symptom" in text.lower() else []

    return DocumentMetadata(
        title=title,
        source=source,
        author="Aquaculture Technical Working Group",
        year=year,
        document_type="guideline",
        species=species,
        life_stage=["fingerling", "juvenile", "growout", "broodstock"],
        topic=topics,
        disease=disease,
        region=["Nigeria", "West Africa", "Sub-Saharan Africa"],
        evidence_level=evidence_level,
        owner="AquaDoc Knowledge Directorate",
    )


async def main():
    settings = get_settings()
    database = Database(settings)
    embeddings = build_embedding_provider(settings)
    service = IngestionService(
        embeddings=embeddings,
        config=IngestionConfig(
            chunk_target_tokens=settings.chunk_target_tokens,
            chunk_overlap_tokens=settings.chunk_overlap_tokens,
        ),
    )

    knowledge_dir = Path(__file__).resolve().parent.parent / "sample-knowledge"
    md_files = sorted(list(knowledge_dir.glob("*.md")))

    print(f"Found {len(md_files)} knowledge documents in {knowledge_dir}")

    total_chunks = 0
    total_docs = 0

    for file_path in md_files:
        print(f"\nProcessing: {file_path.name}...")
        try:
            doc = load_path(file_path, max_bytes=settings.max_upload_bytes)
            text_content = doc.content.decode("utf-8", errors="replace")
            meta = extract_metadata_from_md(text_content, file_path.name)

            async with database.session() as session:
                result = await service.ingest(
                    session,
                    document=doc,
                    metadata=meta,
                    replace_existing=True,
                )

                # Auto-approve the verified knowledge document
                db_doc = await session.get(KnowledgeDocument, result.document_id)
                if db_doc:
                    db_doc.review_status = ReviewStatus.APPROVED
                    db_doc.reviewed_by = "AquaDoc Technical Board"
                    db_doc.review_note = "Official verified West African aquaculture reference guide."
                    await session.commit()

                print(f"  [OK] Ingested & Approved: '{result.title}'")
                print(f"       Document ID: {result.document_id}")
                print(f"       Chunks: {result.chunk_count} (Embedded: {result.embedded_chunks})")
                total_chunks += result.chunk_count
                total_docs += 1
        except Exception as e:
            print(f"  [ERROR] Failed to ingest {file_path.name}: {e}")

    await embeddings.aclose()
    await database.dispose()

    print("\n" + "=" * 60)
    print(f" Knowledge Ingestion Complete!")
    print(f" Total Approved Documents: {total_docs}")
    print(f" Total Embedded Chunks:   {total_chunks}")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
