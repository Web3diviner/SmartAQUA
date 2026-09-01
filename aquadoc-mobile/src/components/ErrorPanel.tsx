/**
 * Failure display.
 *
 * 15_AQUADOC_FRONTEND.md section 15: "Do not collapse all failures into
 * `Something went wrong`." Each failure names what actually broke and what to
 * check, so a retrieval outage is never mistaken for a model outage.
 */

import { AquaDocApiError, type ApiErrorKind } from '@/api/client'

const TITLES: Record<ApiErrorKind, string> = {
  network: 'Cannot reach AquaDoc',
  unauthorized: 'Development token rejected',
  forbidden: 'Developer API disabled',
  not_found: 'Not found',
  validation: 'Request rejected',
  retrieval_failed: 'Knowledge retrieval failed',
  llm_failed: 'Language model unavailable',
  llm_refused: 'Model declined this request',
  response_invalid: 'Invalid response from AquaDoc',
  upload_rejected: 'Upload rejected',
  document_not_approved: 'Document not approved',
  context_incomplete: 'Farm context could not be read',
  timeout: 'AquaDoc timed out',
  server: 'AquaDoc internal error',
}

export function ErrorPanel({
  error,
  onRetry,
  showDetails = false,
}: {
  error: unknown
  onRetry?: () => void
  showDetails?: boolean
}) {
  const apiError = error instanceof AquaDocApiError ? error : null

  const title = apiError ? TITLES[apiError.kind] : 'Unexpected error'
  const guidance = apiError
    ? apiError.guidance
    : 'An unexpected error occurred in the frontend. Check the browser console.'
  const message = apiError ? apiError.message : ''

  // A refusal or a rejected request will fail identically on retry.
  const retryable =
    apiError === null ||
    !['llm_refused', 'validation', 'forbidden', 'unauthorized', 'upload_rejected'].includes(
      apiError.kind,
    )

  return (
    <div className={`error-panel error-panel--${apiError?.kind ?? 'unknown'}`} role="alert">
      <h3 className="error-panel__title">{title}</h3>
      {message && <p className="error-panel__message">{message}</p>}
      <p className="error-panel__guidance">{guidance}</p>

      {apiError?.requestId && (
        <p className="error-panel__request-id">
          Request ID: <code>{apiError.requestId}</code>
        </p>
      )}

      {/* Schema mismatches are only actionable with the failing paths. */}
      {showDetails && apiError && apiError.validationIssues.length > 0 && (
        <details className="error-panel__details">
          <summary>Schema validation issues ({apiError.validationIssues.length})</summary>
          <ul>
            {apiError.validationIssues.map((issue) => (
              <li key={issue}><code>{issue}</code></li>
            ))}
          </ul>
        </details>
      )}

      {onRetry && retryable && (
        <button type="button" className="button" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  )
}
