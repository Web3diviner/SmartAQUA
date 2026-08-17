"""Document loading and checksums.

First stage of the ingestion pipeline (04_AQUADOC_RAG_LLM.md section 4):

    approved document -> checksum -> parse -> clean -> preserve page/section
                      -> chunk -> metadata -> embedding -> pgvector -> review

Upload handling follows 07_SECURITY_ARCHITECTURE.md section 9: MIME allowlist,
content-sniffed type, size limits, and generated storage names. A user-supplied
filename never becomes a filesystem path.
"""

from __future__ import annotations

import hashlib
import uuid
from dataclasses import dataclass
from pathlib import Path

from app.errors import UploadRejectedError

#: Extension -> canonical MIME type. Anything not listed is rejected.
ALLOWED_TYPES: dict[str, str] = {
    ".pdf": "application/pdf",
    ".txt": "text/plain",
    ".md": "text/markdown",
}

#: Leading bytes used to verify the declared type actually matches the content.
_MAGIC_BYTES: dict[str, bytes] = {
    "application/pdf": b"%PDF-",
}


@dataclass(frozen=True)
class LoadedDocument:
    """Raw bytes plus everything needed to store them safely."""

    #: Random storage name. Never derived from user input.
    storage_name: str
    #: Original name, for display only.
    display_name: str
    media_type: str
    content: bytes
    checksum: str

    @property
    def size_bytes(self) -> int:
        return len(self.content)


def compute_checksum(content: bytes) -> str:
    """SHA-256 over the raw bytes.

    Identifies a document version so re-ingesting the same file is detectable
    (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 9: version/checksum required).
    """
    return hashlib.sha256(content).hexdigest()


def load_bytes(
    content: bytes,
    *,
    filename: str,
    max_bytes: int,
) -> LoadedDocument:
    """Validate and wrap an uploaded file.

    Raises:
        UploadRejectedError: empty, oversized, unsupported type, or a declared
            type that does not match the actual content.
    """
    if not content:
        raise UploadRejectedError("The uploaded file is empty.")
    if len(content) > max_bytes:
        raise UploadRejectedError(
            f"The uploaded file is {len(content)} bytes, above the "
            f"{max_bytes}-byte limit."
        )

    # Take only the extension from the user-supplied name; discard any path.
    suffix = Path(filename).suffix.lower()
    media_type = ALLOWED_TYPES.get(suffix)
    if media_type is None:
        raise UploadRejectedError(
            f"Unsupported file type '{suffix or 'unknown'}'. "
            f"Allowed: {', '.join(sorted(ALLOWED_TYPES))}."
        )

    # Content sniffing: a .pdf that is not a PDF is rejected here rather than
    # failing confusingly in the parser.
    magic = _MAGIC_BYTES.get(media_type)
    if magic is not None and not content.startswith(magic):
        raise UploadRejectedError(
            f"File content does not match its '{suffix}' extension."
        )

    return LoadedDocument(
        storage_name=f"{uuid.uuid4().hex}{suffix}",
        display_name=Path(filename).name[:255],
        media_type=media_type,
        content=content,
        checksum=compute_checksum(content),
    )


def load_path(path: Path, *, max_bytes: int) -> LoadedDocument:
    """Load a document from disk. For the CLI ingestion tool."""
    if not path.is_file():
        raise UploadRejectedError(f"No such file: {path}")
    return load_bytes(path.read_bytes(), filename=path.name, max_bytes=max_bytes)
