"""Typed application errors.

15_AQUADOC_FRONTEND.md section 15 requires the frontend to distinguish
retrieval failure from LLM failure from validation failure. That is only
possible if the service emits distinct, stable error codes — so every failure
mode below carries its own code rather than collapsing into a generic 500.

Wire format is fixed by 05_API_AND_SERVICE_CONTRACTS.md:

    {"error": {"code": "...", "message": "...", "request_id": "..."}}
"""

from __future__ import annotations

from typing import Any


class AquaDocError(Exception):
    """Base class for errors that map to a stable API error code."""

    code = "INTERNAL_ERROR"
    status_code = 500
    # Message shown to the caller. Never include stack traces, prompts, or secrets
    # (15_AQUADOC_FRONTEND.md section 10; 07_SECURITY_ARCHITECTURE.md section 10).
    public_message = "An unexpected error occurred."

    def __init__(self, message: str | None = None, *, details: dict[str, Any] | None = None) -> None:
        super().__init__(message or self.public_message)
        self.message = message or self.public_message
        self.details = details or {}


# -- Client errors -----------------------------------------------------------


class ValidationError(AquaDocError):
    code = "VALIDATION_ERROR"
    status_code = 422
    public_message = "The request was not valid."


class NotFoundError(AquaDocError):
    code = "NOT_FOUND"
    status_code = 404
    public_message = "The requested resource was not found."


class DocumentNotFoundError(NotFoundError):
    code = "DOCUMENT_NOT_FOUND"
    public_message = "Knowledge document was not found."


class ConversationNotFoundError(NotFoundError):
    code = "CONVERSATION_NOT_FOUND"
    public_message = "Conversation was not found."


class RetrievalTraceNotFoundError(NotFoundError):
    code = "RETRIEVAL_TRACE_NOT_FOUND"
    public_message = "No retrieval trace was recorded for that request ID."


class UnauthorizedError(AquaDocError):
    code = "UNAUTHORIZED"
    status_code = 401
    public_message = "Authentication is required."


class ForbiddenError(AquaDocError):
    code = "FORBIDDEN"
    status_code = 403
    public_message = "This operation is not permitted."


class UploadRejectedError(AquaDocError):
    code = "UPLOAD_REJECTED"
    status_code = 400
    public_message = "The uploaded file was rejected."


class DocumentNotApprovedError(AquaDocError):
    code = "DOCUMENT_NOT_APPROVED"
    status_code = 409
    public_message = "The knowledge document is not approved for production retrieval."


class ParseError(AquaDocError):
    """The document could not be turned into text.

    A scanned PDF with no OCR layer is the common case; it is a property of the
    file, not a server fault, so this is a 400.
    """

    code = "DOCUMENT_PARSE_FAILED"
    status_code = 400
    public_message = "The document could not be parsed."


# -- Dependency / pipeline errors -------------------------------------------


class ContextIncompleteError(AquaDocError):
    code = "CONTEXT_INCOMPLETE"
    status_code = 422
    public_message = "The supplied farm context could not be interpreted."


class EmbeddingProviderError(AquaDocError):
    code = "EMBEDDING_PROVIDER_FAILED"
    status_code = 502
    public_message = "The embedding provider could not be reached."


class RetrievalError(AquaDocError):
    code = "RETRIEVAL_FAILED"
    status_code = 503
    public_message = "Knowledge retrieval failed."


class LLMProviderError(AquaDocError):
    code = "LLM_PROVIDER_FAILED"
    status_code = 502
    public_message = "The language model provider could not be reached."


class LLMRefusalError(AquaDocError):
    """The provider's safety classifiers declined the request.

    Distinct from a transport failure: retrying the same prompt will not help.
    """

    code = "LLM_REFUSED"
    status_code = 422
    public_message = "The request could not be answered by the language model."


class ResponseValidationError(AquaDocError):
    """The model returned something that does not satisfy the response schema.

    15_AQUADOC_FRONTEND.md section 11: invalid responses must fail safely rather
    than being rendered as if they were valid.
    """

    code = "RESPONSE_VALIDATION_FAILED"
    status_code = 502
    public_message = "The language model returned a response in an unexpected shape."


class DatabaseUnavailableError(AquaDocError):
    code = "DATABASE_UNAVAILABLE"
    status_code = 503
    public_message = "The knowledge database is unavailable."


# -- FastAPI integration -----------------------------------------------------


def _error_body(code: str, message: str, request_id: str, details: dict[str, Any]) -> dict[str, Any]:
    """The envelope fixed by 05_API_AND_SERVICE_CONTRACTS.md."""
    payload: dict[str, Any] = {"code": code, "message": message, "request_id": request_id}
    if details:
        payload["details"] = details
    return {"error": payload}


def register_exception_handlers(app: Any) -> None:
    """Install handlers so every failure returns the same envelope.

    Three properties matter here:

    - `request_id` is always present, so a farmer's report of "it failed" is
      traceable to a log line.
    - Codes are stable and specific, so the client can distinguish a retrieval
      failure from an LLM failure (15_AQUADOC_FRONTEND.md section 15).
    - Unhandled exceptions never leak their message. A stack trace or driver
      error can contain connection strings or prompt fragments
      (07_SECURITY_ARCHITECTURE.md section 10), so those are logged server-side
      and reported as a generic 500.
    """
    import logging

    from fastapi import Request
    from fastapi.exceptions import RequestValidationError
    from fastapi.responses import JSONResponse
    from starlette.exceptions import HTTPException as StarletteHTTPException

    logger = logging.getLogger(__name__)

    def _request_id(request: Request) -> str:
        return getattr(request.state, "request_id", "-")

    @app.exception_handler(AquaDocError)
    async def _handle_aquadoc_error(request: Request, exc: AquaDocError) -> JSONResponse:
        # Server-side failures are logged; client mistakes are not worth the noise.
        if exc.status_code >= 500:
            logger.error(
                "aquadoc_error",
                extra={"code": exc.code, "status_code": exc.status_code},
                exc_info=exc,
            )
        return JSONResponse(
            status_code=exc.status_code,
            content=_error_body(exc.code, exc.message, _request_id(request), exc.details),
        )

    @app.exception_handler(RequestValidationError)
    async def _handle_validation_error(
        request: Request, exc: RequestValidationError
    ) -> JSONResponse:
        # Pydantic's `input` field echoes the submitted value; it is dropped so
        # farmer-supplied text is not reflected back in an error body.
        details = {
            "fields": [
                {"location": list(error.get("loc", [])), "message": error.get("msg", "")}
                for error in exc.errors()
            ]
        }
        return JSONResponse(
            status_code=ValidationError.status_code,
            content=_error_body(
                ValidationError.code,
                "The request did not match the expected schema.",
                _request_id(request),
                details,
            ),
        )

    @app.exception_handler(StarletteHTTPException)
    async def _handle_http_exception(
        request: Request, exc: StarletteHTTPException
    ) -> JSONResponse:
        # Registered against Starlette's class, not FastAPI's subclass, so that
        # router-raised 404s and 405s use the same envelope as everything else.
        codes = {401: "UNAUTHORIZED", 403: "FORBIDDEN", 404: "NOT_FOUND", 405: "METHOD_NOT_ALLOWED"}
        return JSONResponse(
            status_code=exc.status_code,
            content=_error_body(
                codes.get(exc.status_code, "HTTP_ERROR"),
                str(exc.detail),
                _request_id(request),
                {},
            ),
            headers=getattr(exc, "headers", None),
        )

    @app.exception_handler(Exception)
    async def _handle_unexpected(request: Request, exc: Exception) -> JSONResponse:
        logger.exception("unhandled_exception", extra={"path": request.url.path})
        return JSONResponse(
            status_code=500,
            content=_error_body(
                AquaDocError.code,
                AquaDocError.public_message,
                _request_id(request),
                {},
            ),
        )
