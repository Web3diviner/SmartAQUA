"""Parsing: bytes -> page-and-section-aware text.

04_AQUADOC_RAG_LLM.md section 4 requires page and section structure to be
preserved through parsing, because citations must be able to point at a page
(15_AQUADOC_FRONTEND.md section 12). Flattening a PDF to one string loses that
permanently, so structure is carried from here to the chunker.
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass

from app.errors import ParseError

logger = logging.getLogger(__name__)

#: Markdown ATX headings become section labels.
_MD_HEADING = re.compile(r"^(#{1,4})\s+(.+?)\s*#*$", re.MULTILINE)

#: A numbered heading in plain text, e.g. "3.2 Water Quality Management".
_NUMBERED_HEADING = re.compile(r"^\s*(\d+(?:\.\d+){0,3})\s+([A-Z][^\n]{2,80})$", re.MULTILINE)


@dataclass(frozen=True)
class ParsedBlock:
    """A run of text with its position in the source document."""

    text: str
    #: 1-based. `None` for formats without pages.
    page_number: int | None
    #: Nearest enclosing heading, if one was found.
    section: str | None


@dataclass(frozen=True)
class ParsedDocument:
    blocks: list[ParsedBlock]
    page_count: int | None

    @property
    def is_empty(self) -> bool:
        return not any(block.text.strip() for block in self.blocks)


def parse(content: bytes, media_type: str) -> ParsedDocument:
    if media_type == "application/pdf":
        return _parse_pdf(content)
    if media_type in {"text/plain", "text/markdown"}:
        return _parse_text(content, markdown=media_type == "text/markdown")
    raise ParseError(f"No parser registered for media type '{media_type}'.")


def _parse_pdf(content: bytes) -> ParsedDocument:
    try:
        from pypdf import PdfReader
    except ImportError as exc:  # pragma: no cover - dependency guard
        raise ParseError("The 'pypdf' package is required to parse PDF documents.") from exc

    import io

    try:
        reader = PdfReader(io.BytesIO(content))
    except Exception as exc:
        raise ParseError("The PDF could not be opened; it may be corrupt or encrypted.") from exc

    blocks: list[ParsedBlock] = []
    current_section: str | None = None

    for index, page in enumerate(reader.pages, start=1):
        try:
            text = page.extract_text() or ""
        except Exception:
            # One unreadable page should not fail the whole document; the gap is
            # logged and the remaining pages still ingest.
            logger.warning("pdf_page_extraction_failed", extra={"page": index})
            continue

        # Track the last heading seen so a chunk mid-section still knows where
        # it sits, even when the heading was on an earlier page.
        headings = _NUMBERED_HEADING.findall(text)
        if headings:
            number, title = headings[-1]
            current_section = f"{number} {title}".strip()

        blocks.append(ParsedBlock(text=text, page_number=index, section=current_section))

    if not blocks:
        raise ParseError(
            "No extractable text was found. The PDF may be a scan without an "
            "OCR text layer."
        )
    return ParsedDocument(blocks=blocks, page_count=len(reader.pages))


def _parse_text(content: bytes, *, markdown: bool) -> ParsedDocument:
    try:
        text = content.decode("utf-8")
    except UnicodeDecodeError:
        # Fall back rather than fail: legacy exports are often latin-1.
        text = content.decode("latin-1", errors="replace")

    if not markdown:
        return ParsedDocument(
            blocks=[ParsedBlock(text=text, page_number=None, section=None)],
            page_count=None,
        )

    # Split on headings so each block carries its own section label.
    blocks: list[ParsedBlock] = []
    matches = list(_MD_HEADING.finditer(text))

    if not matches:
        return ParsedDocument(
            blocks=[ParsedBlock(text=text, page_number=None, section=None)],
            page_count=None,
        )

    preamble = text[: matches[0].start()].strip()
    if preamble:
        blocks.append(ParsedBlock(text=preamble, page_number=None, section=None))

    for index, match in enumerate(matches):
        heading = match.group(2).strip()
        end = matches[index + 1].start() if index + 1 < len(matches) else len(text)
        body = text[match.end() : end].strip()
        if body:
            blocks.append(ParsedBlock(text=body, page_number=None, section=heading))

    return ParsedDocument(blocks=blocks, page_count=None)
