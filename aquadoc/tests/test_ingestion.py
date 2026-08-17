"""Ingestion: upload validation, cleaning, chunking, and governance."""

from __future__ import annotations

import pytest

from app.errors import ParseError, UploadRejectedError
from app.schemas.common import EvidenceLevel
from app.schemas.knowledge import DocumentMetadata
from ingestion.chunker import Chunker, estimate_tokens
from ingestion.cleaner import clean_text, find_repeated_lines, strip_repeated_lines
from ingestion.loader import compute_checksum, load_bytes
from ingestion.parser import ParsedBlock, parse


class TestUploadValidation:
    """07_SECURITY_ARCHITECTURE.md section 9."""

    def test_storage_name_is_generated_not_user_supplied(self) -> None:
        """A user-controlled filename must never become a filesystem path."""
        document = load_bytes(
            b"hello world",
            filename="../../etc/passwd.txt",
            max_bytes=1024,
        )

        assert "/" not in document.storage_name
        assert ".." not in document.storage_name
        assert document.storage_name.endswith(".txt")
        # The original is kept for display only.
        assert document.display_name == "passwd.txt"

    def test_oversized_upload_rejected(self) -> None:
        with pytest.raises(UploadRejectedError):
            load_bytes(b"x" * 2048, filename="big.txt", max_bytes=1024)

    def test_empty_upload_rejected(self) -> None:
        with pytest.raises(UploadRejectedError):
            load_bytes(b"", filename="empty.txt", max_bytes=1024)

    def test_unsupported_type_rejected(self) -> None:
        with pytest.raises(UploadRejectedError, match="Unsupported file type"):
            load_bytes(b"MZ\x90\x00", filename="malware.exe", max_bytes=1024)

    def test_content_must_match_declared_extension(self) -> None:
        """A .pdf that is not a PDF is caught here, not deep in the parser."""
        with pytest.raises(UploadRejectedError, match="does not match"):
            load_bytes(b"not a pdf at all", filename="fake.pdf", max_bytes=1024)

    def test_checksum_is_stable_and_content_addressed(self) -> None:
        first = load_bytes(b"same content", filename="a.txt", max_bytes=1024)
        second = load_bytes(b"same content", filename="b.txt", max_bytes=1024)

        assert first.checksum == second.checksum
        assert first.checksum == compute_checksum(b"same content")
        # Different storage names, so re-upload never overwrites.
        assert first.storage_name != second.storage_name


class TestCleaning:
    def test_hyphenated_line_breaks_are_rejoined(self) -> None:
        assert "dissolved" in clean_text("The dissol-\nved oxygen level")

    def test_page_number_lines_removed(self) -> None:
        cleaned = clean_text("Content here.\n\n42\n\nMore content.")

        assert "Content here." in cleaned
        assert "More content." in cleaned
        assert "\n42\n" not in cleaned

    def test_numbers_and_units_are_never_altered(self) -> None:
        """Corrupting a measurement would silently corrupt guidance."""
        text = "Maintain dissolved oxygen above 5.0 mg/L and pH between 6.5 and 8.5."
        cleaned = clean_text(text)

        for token in ("5.0 mg/L", "6.5", "8.5"):
            assert token in cleaned

    def test_running_headers_detected_across_pages(self) -> None:
        pages = [f"FAO Aquaculture Manual\nBody text for page {i}.\n" for i in range(10)]
        repeated = find_repeated_lines(pages)

        assert "FAO Aquaculture Manual" in repeated
        assert strip_repeated_lines(pages[0], repeated).strip() == "Body text for page 0."

    def test_short_documents_do_not_trigger_header_detection(self) -> None:
        """With few pages, repetition is not evidence of chrome."""
        assert find_repeated_lines(["Same line", "Same line"]) == set()


class TestChunking:
    """04_AQUADOC_RAG_LLM.md section 6."""

    def test_short_text_stays_one_chunk(self) -> None:
        blocks = [ParsedBlock(text="A short passage about feeding.", page_number=1, section=None)]
        chunks = Chunker(target_tokens=750, overlap_tokens=150).chunk_blocks(blocks)

        assert len(chunks) == 1

    def test_long_text_splits_near_the_target(self) -> None:
        paragraph = "Aquaculture management requires careful monitoring. " * 40
        blocks = [ParsedBlock(text=paragraph, page_number=1, section=None)]
        chunks = Chunker(target_tokens=100, overlap_tokens=20).chunk_blocks(blocks)

        assert len(chunks) > 1
        # Allow headroom for the overlap prefix carried into each chunk.
        assert all(chunk.token_estimate <= 100 * 2 for chunk in chunks)

    def test_page_and_section_survive_chunking(self) -> None:
        """Citations must be able to point at a page."""
        blocks = [
            ParsedBlock(text="Feeding content. " * 60, page_number=7, section="3.2 Feeding")
        ]
        chunks = Chunker(target_tokens=50, overlap_tokens=10).chunk_blocks(blocks)

        assert len(chunks) > 1
        assert all(chunk.page_number == 7 for chunk in chunks)
        assert all(chunk.section == "3.2 Feeding" for chunk in chunks)

    def test_chunks_never_span_two_sections(self) -> None:
        """Keeps unrelated topics out of a single chunk."""
        blocks = [
            ParsedBlock(text="Feeding text.", page_number=1, section="Feeding"),
            ParsedBlock(text="Disease text.", page_number=2, section="Disease"),
        ]
        chunks = Chunker(target_tokens=750, overlap_tokens=150).chunk_blocks(blocks)

        assert len(chunks) == 2
        assert chunks[0].section == "Feeding"
        assert chunks[1].section == "Disease"

    def test_chunk_indices_are_contiguous(self) -> None:
        blocks = [ParsedBlock(text="Content. " * 100, page_number=1, section=None)]
        chunks = Chunker(target_tokens=40, overlap_tokens=8).chunk_blocks(blocks)

        assert [chunk.index for chunk in chunks] == list(range(len(chunks)))

    def test_overlap_is_configurable_and_bounded(self) -> None:
        with pytest.raises(ValueError, match="overlap"):
            Chunker(target_tokens=100, overlap_tokens=100)

    def test_token_estimate_is_positive(self) -> None:
        assert estimate_tokens("") >= 1
        assert estimate_tokens("a" * 400) == 100


class TestParsing:
    def test_markdown_headings_become_sections(self) -> None:
        content = b"# Water Quality\n\nKeep oxygen above 5 mg/L.\n\n# Feeding\n\nFeed twice daily."
        parsed = parse(content, "text/markdown")

        sections = [block.section for block in parsed.blocks]
        assert "Water Quality" in sections
        assert "Feeding" in sections

    def test_plain_text_has_no_page_or_section(self) -> None:
        parsed = parse(b"Just some text.", "text/plain")

        assert parsed.blocks[0].page_number is None
        assert parsed.blocks[0].section is None

    def test_unknown_media_type_raises(self) -> None:
        with pytest.raises(ParseError):
            parse(b"data", "application/octet-stream")

    def test_non_utf8_text_still_parses(self) -> None:
        """Legacy exports are often latin-1; a decode error should not fail ingest."""
        parsed = parse("Température de l'eau".encode("latin-1"), "text/plain")

        assert parsed.blocks[0].text


class TestMetadataContract:
    def test_evidence_level_is_required(self) -> None:
        """04_AQUADOC_RAG_LLM.md section 5 makes evidence level mandatory."""
        with pytest.raises(Exception):
            DocumentMetadata(title="T", source="S", document_type="guideline")

    def test_metadata_accepts_the_documented_fields(self) -> None:
        metadata = DocumentMetadata(
            title="FAO Aquaculture Manual",
            source="FAO",
            author="FAO Fisheries",
            year=2022,
            document_type="guideline",
            species=["Clarias gariepinus"],
            life_stage=["grow_out"],
            topic=["feeding", "water_quality"],
            disease=[],
            region=["West Africa"],
            evidence_level=EvidenceLevel.A_OFFICIAL_GUIDELINE,
            owner="knowledge-team",
        )

        assert metadata.evidence_level is EvidenceLevel.A_OFFICIAL_GUIDELINE
        assert metadata.topic == ["feeding", "water_quality"]
