"""Chunking.

04_AQUADOC_RAG_LLM.md section 6 initial configuration:

  - target chunk size 600-900 tokens
  - overlap 100-200 tokens
  - preserve headings
  - do not split tables blindly
  - keep source page
  - avoid chunks containing multiple unrelated topics

The doc also says to tune these with retrieval evaluation rather than guessing
permanently, so every parameter is a constructor argument.

Splitting is hierarchical: paragraph boundaries first, then sentences, then
words. Cutting mid-sentence is a last resort — a chunk that ends mid-clause
embeds poorly and reads badly as a citation excerpt.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

from ingestion.parser import ParsedBlock

#: Rough token estimate: ~4 characters per token for English prose. Deliberately
#: approximate — chunk sizing does not need tokenizer-exact counts, and a real
#: tokenizer would tie chunking to one model's vocabulary.
CHARS_PER_TOKEN = 4

_PARAGRAPH_BREAK = re.compile(r"\n\s*\n")
_SENTENCE_END = re.compile(r"(?<=[.!?])\s+(?=[A-Z(\[])")

#: A line that looks like a table row: multiple columns separated by runs of
#: whitespace or pipes. Tables are kept whole where possible.
_TABLE_ROW = re.compile(r"^\s*\S+(?:\s{2,}|\s*\|\s*)\S+.*$")

#: Consecutive table-like lines needed before a block is treated as a table.
_TABLE_MIN_ROWS = 3


def estimate_tokens(text: str) -> int:
    return max(1, len(text) // CHARS_PER_TOKEN)


@dataclass(frozen=True)
class Chunk:
    """One retrievable unit of a document."""

    index: int
    content: str
    token_estimate: int
    page_number: int | None
    section: str | None


class Chunker:
    def __init__(self, *, target_tokens: int = 750, overlap_tokens: int = 150) -> None:
        if overlap_tokens >= target_tokens:
            raise ValueError("overlap_tokens must be smaller than target_tokens")
        self._target_chars = target_tokens * CHARS_PER_TOKEN
        self._overlap_chars = overlap_tokens * CHARS_PER_TOKEN
        #: Below this, a trailing fragment is merged back rather than kept as
        #: its own chunk — a 30-token orphan retrieves noise.
        self._min_chars = max(120, self._target_chars // 6)

    def chunk_blocks(self, blocks: list[ParsedBlock]) -> list[Chunk]:
        """Chunk a parsed document, preserving page and section attribution.

        Blocks are chunked independently so a chunk never spans two sections —
        that is what keeps unrelated topics out of a single chunk.
        """
        chunks: list[Chunk] = []
        for block in blocks:
            text = block.text.strip()
            if not text:
                continue
            for piece in self._split(text):
                chunks.append(
                    Chunk(
                        index=len(chunks),
                        content=piece,
                        token_estimate=estimate_tokens(piece),
                        page_number=block.page_number,
                        section=block.section,
                    )
                )
        return chunks

    # -- splitting -----------------------------------------------------------

    def _split(self, text: str) -> list[str]:
        if len(text) <= self._target_chars:
            return [text]

        units = self._segment(text)
        chunks: list[str] = []
        current = ""

        for unit in units:
            candidate = f"{current}\n\n{unit}" if current else unit

            if len(candidate) <= self._target_chars:
                current = candidate
                continue

            if current:
                chunks.append(current)
                current = self._carry_overlap(current, unit)
            else:
                # A single unit larger than the target (a long table or an
                # unbroken paragraph): split it directly.
                pieces = self._hard_split(unit)
                chunks.extend(pieces[:-1])
                current = pieces[-1]

        if current:
            chunks.append(current)
        return self._merge_orphan(chunks)

    def _segment(self, text: str) -> list[str]:
        """Break text into paragraph-level units, keeping tables intact."""
        units: list[str] = []
        for paragraph in _PARAGRAPH_BREAK.split(text):
            paragraph = paragraph.strip()
            if not paragraph:
                continue
            if self._looks_like_table(paragraph) or len(paragraph) <= self._target_chars:
                units.append(paragraph)
            else:
                units.extend(self._split_sentences(paragraph))
        return units

    def _split_sentences(self, paragraph: str) -> list[str]:
        sentences = _SENTENCE_END.split(paragraph)
        units: list[str] = []
        buffer = ""
        for sentence in sentences:
            candidate = f"{buffer} {sentence}".strip() if buffer else sentence
            if len(candidate) <= self._target_chars:
                buffer = candidate
            else:
                if buffer:
                    units.append(buffer)
                buffer = sentence
        if buffer:
            units.append(buffer)
        return units

    def _hard_split(self, text: str) -> list[str]:
        """Last resort: split on word boundaries at the target size."""
        pieces: list[str] = []
        words = text.split(" ")
        buffer = ""
        for word in words:
            candidate = f"{buffer} {word}".strip() if buffer else word
            if len(candidate) <= self._target_chars:
                buffer = candidate
            else:
                if buffer:
                    pieces.append(buffer)
                buffer = word
        if buffer:
            pieces.append(buffer)
        return pieces or [text]

    def _carry_overlap(self, previous: str, next_unit: str) -> str:
        """Start the next chunk with the tail of the previous one.

        Overlap keeps a fact that straddles a boundary retrievable from either
        side. The tail is cut on a sentence boundary where one exists nearby.
        """
        if self._overlap_chars <= 0:
            return next_unit

        tail = previous[-self._overlap_chars :]
        boundary = _SENTENCE_END.search(tail)
        if boundary:
            tail = tail[boundary.end() :]
        tail = tail.strip()
        return f"{tail}\n\n{next_unit}" if tail else next_unit

    def _merge_orphan(self, chunks: list[str]) -> list[str]:
        """Fold a too-small trailing chunk into its predecessor."""
        if len(chunks) >= 2 and len(chunks[-1]) < self._min_chars:
            merged = f"{chunks[-2]}\n\n{chunks[-1]}"
            return [*chunks[:-2], merged]
        return chunks

    @staticmethod
    def _looks_like_table(text: str) -> bool:
        lines = [line for line in text.split("\n") if line.strip()]
        if len(lines) < _TABLE_MIN_ROWS:
            return False
        matches = sum(1 for line in lines if _TABLE_ROW.match(line))
        return matches >= _TABLE_MIN_ROWS and matches / len(lines) >= 0.6
