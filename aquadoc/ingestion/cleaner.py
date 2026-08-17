"""Cleaning: normalise extracted text without destroying meaning.

PDF text extraction introduces artefacts that hurt both embedding quality and
readability of citation excerpts: hyphens splitting words across line breaks,
repeated running headers, page-number lines, and hard-wrapped paragraphs.

The rule applied throughout: remove layout noise, never remove content. Numbers
and units in particular are left exactly as written, because a mangled "6.5 mg/L"
would silently corrupt aquaculture guidance.
"""

from __future__ import annotations

import re
from collections import Counter
from collections.abc import Iterable

#: A word split across a line break: "dissol-\nved" -> "dissolved".
_HYPHEN_LINEBREAK = re.compile(r"(\w)-\s*\n\s*(\w)")

#: A line containing only a page number, optionally decorated.
_PAGE_NUMBER_LINE = re.compile(r"^\s*[-–—|]*\s*(?:page\s+)?\d{1,4}\s*[-–—|]*\s*$", re.IGNORECASE)

#: Three or more blank lines collapse to a paragraph break.
_EXCESS_BLANK_LINES = re.compile(r"\n{3,}")

#: Runs of spaces and tabs, but not newlines.
_HORIZONTAL_WHITESPACE = re.compile(r"[ \t ]{2,}")

#: Control characters that survive extraction and break JSON round-trips.
_CONTROL_CHARS = re.compile(r"[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]")

#: Ligatures and typographic characters normalised for consistent tokenisation.
_REPLACEMENTS = {
    "ﬀ": "ff",
    "ﬁ": "fi",
    "ﬂ": "fl",
    "ﬃ": "ffi",
    "ﬄ": "ffl",
    "‘": "'",
    "’": "'",
    "“": '"',
    "”": '"',
    "–": "-",
    "—": "-",
    "­": "",  # soft hyphen
}

#: A line must repeat on at least this share of pages to count as a header.
_REPEAT_THRESHOLD = 0.6

#: Only short lines are candidates for running headers.
_MAX_HEADER_LENGTH = 90


def clean_text(text: str) -> str:
    """Normalise one block of extracted text."""
    text = _CONTROL_CHARS.sub("", text)
    for source, target in _REPLACEMENTS.items():
        text = text.replace(source, target)

    # Rejoin hyphen-split words before any line-level work.
    text = _HYPHEN_LINEBREAK.sub(r"\1\2", text)

    lines = [line for line in text.split("\n") if not _PAGE_NUMBER_LINE.match(line)]
    text = "\n".join(line.rstrip() for line in lines)

    text = _HORIZONTAL_WHITESPACE.sub(" ", text)
    text = _EXCESS_BLANK_LINES.sub("\n\n", text)
    return text.strip()


def find_repeated_lines(page_texts: Iterable[str]) -> set[str]:
    """Identify running headers and footers across pages.

    A short line appearing on most pages is chrome, not content. Detecting it
    statistically avoids hardcoding any single publisher's layout.
    """
    pages = list(page_texts)
    if len(pages) < 4:
        # Too few pages for the repetition signal to be trustworthy.
        return set()

    counts: Counter[str] = Counter()
    for page in pages:
        # Headers and footers live in the first and last few lines.
        lines = [line.strip() for line in page.split("\n") if line.strip()]
        for line in lines[:3] + lines[-3:]:
            if len(line) <= _MAX_HEADER_LENGTH:
                counts[line] += 1

    minimum = max(3, int(len(pages) * _REPEAT_THRESHOLD))
    return {line for line, count in counts.items() if count >= minimum}


def strip_repeated_lines(text: str, repeated: set[str]) -> str:
    """Remove previously identified running headers and footers."""
    if not repeated:
        return text
    kept = [line for line in text.split("\n") if line.strip() not in repeated]
    return _EXCESS_BLANK_LINES.sub("\n\n", "\n".join(kept)).strip()
