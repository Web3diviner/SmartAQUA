/**
 * Centralised API client.
 *
 * 15_AQUADOC_FRONTEND.md section 6 requires a single client rather than fetch
 * calls scattered through components, and section 15 requires distinguishable
 * failure states:
 *
 *   - frontend cannot reach AquaDoc
 *   - RAG retrieval failed
 *   - LLM provider failed
 *   - response validation failed
 *   - upload failed
 *   - knowledge document not approved
 *   - context incomplete
 *
 * Those map to `AquaDocApiError.kind`, so the UI can say what actually broke
 * instead of "Something went wrong".
 */

import { z } from 'zod'
import { errorResponseSchema } from '@/schemas/aquadoc'

export type ApiErrorKind =
  | 'network'
  | 'unauthorized'
  | 'forbidden'
  | 'not_found'
  | 'validation'
  | 'retrieval_failed'
  | 'llm_failed'
  | 'llm_refused'
  | 'response_invalid'
  | 'upload_rejected'
  | 'document_not_approved'
  | 'context_incomplete'
  | 'timeout'
  | 'server'

/** Service error codes mapped to the UI's failure taxonomy. */
const CODE_TO_KIND: Record<string, ApiErrorKind> = {
  UNAUTHORIZED: 'unauthorized',
  FORBIDDEN: 'forbidden',
  NOT_FOUND: 'not_found',
  DOCUMENT_NOT_FOUND: 'not_found',
  CONVERSATION_NOT_FOUND: 'not_found',
  RETRIEVAL_TRACE_NOT_FOUND: 'not_found',
  VALIDATION_ERROR: 'validation',
  RETRIEVAL_FAILED: 'retrieval_failed',
  EMBEDDING_PROVIDER_FAILED: 'retrieval_failed',
  DATABASE_UNAVAILABLE: 'retrieval_failed',
  LLM_PROVIDER_FAILED: 'llm_failed',
  LLM_REFUSED: 'llm_refused',
  RESPONSE_VALIDATION_FAILED: 'response_invalid',
  UPLOAD_REJECTED: 'upload_rejected',
  DOCUMENT_PARSE_FAILED: 'upload_rejected',
  DOCUMENT_NOT_APPROVED: 'document_not_approved',
  CONTEXT_INCOMPLETE: 'context_incomplete',
}

/** What each failure means, in words a developer can act on. */
const KIND_GUIDANCE: Record<ApiErrorKind, string> = {
  network:
    'The frontend could not reach AquaDoc. Check that the service is running on the configured URL.',
  unauthorized:
    'AquaDoc rejected the development token. Check VITE_AQUADOC_DEV_TOKEN matches AQUADOC_DEV_TOKEN.',
  forbidden:
    'The developer API is disabled on this instance. It requires APP_ENV=development and a configured dev token.',
  not_found: 'The requested resource does not exist.',
  validation: 'The request was rejected as invalid before reaching the pipeline.',
  retrieval_failed:
    'Knowledge retrieval failed — this is a database or embedding problem, not a model problem.',
  llm_failed:
    'The language model provider could not be reached. Retrieval may still be healthy.',
  llm_refused:
    'The language model declined to answer this request. Retrying the same wording will not help.',
  response_invalid:
    'AquaDoc returned a response that did not match the expected schema. The answer was discarded rather than displayed.',
  upload_rejected: 'The document was rejected during upload validation.',
  document_not_approved:
    'The document is not approved, so it is excluded from production retrieval.',
  context_incomplete: 'The supplied farm context could not be interpreted.',
  timeout: 'AquaDoc did not respond in time. Long answers at high effort can take a while.',
  server: 'AquaDoc reported an unexpected internal error.',
}

export class AquaDocApiError extends Error {
  readonly kind: ApiErrorKind
  readonly code: string
  readonly requestId: string | null
  readonly guidance: string
  /** Present when the failure was schema validation, for the debug panel. */
  readonly validationIssues: string[]

  constructor(params: {
    kind: ApiErrorKind
    code: string
    message: string
    requestId?: string | null
    validationIssues?: string[]
  }) {
    super(params.message)
    this.name = 'AquaDocApiError'
    this.kind = params.kind
    this.code = params.code
    this.requestId = params.requestId ?? null
    this.guidance = KIND_GUIDANCE[params.kind]
    this.validationIssues = params.validationIssues ?? []
  }
}

export interface ClientConfig {
  baseUrl: string
  devToken: string
  timeoutMs: number
}

export function readClientConfig(): ClientConfig {
  let defaultBase = 'https://aquadoc-api.onrender.com'
  if (
    typeof window !== 'undefined' &&
    (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')
  ) {
    defaultBase = 'http://localhost:8001'
  }

  const rawEnv =
    import.meta.env.VITE_AQUADOC_BASE_URL ||
    import.meta.env.VITE_AQUADOC_API_URL ||
    defaultBase

  const isProductionBrowser =
    typeof window !== 'undefined' &&
    window.location.hostname !== 'localhost' &&
    window.location.hostname !== '127.0.0.1'

  const resolvedBaseUrl =
    isProductionBrowser && (rawEnv.includes('localhost') || rawEnv.includes('127.0.0.1'))
      ? defaultBase
      : rawEnv

  return {
    baseUrl: resolvedBaseUrl.replace(/\/$/, ''),
    devToken: import.meta.env.VITE_AQUADOC_DEV_TOKEN ?? 'aqua-dev-token-2026',
    // Generous: a farm-assessment answer at high effort legitimately takes time.
    timeoutMs: 120_000,
  }
}

interface RequestOptions<S extends z.ZodTypeAny> {
  path: string
  schema: S
  method?: 'GET' | 'POST'
  body?: unknown
  signal?: AbortSignal
}

/**
 * Perform a request and validate the response against a schema.
 *
 * Generic over the schema rather than its output type: schemas here use
 * `.default()`, so a Zod schema's input and output types differ and a
 * `ZodType<T>` annotation would not accept them.
 *
 * Validation is not optional: an unvalidated payload rendered as advice is the
 * failure 15_AQUADOC_FRONTEND.md section 11 exists to prevent.
 */
export async function request<S extends z.ZodTypeAny>(
  config: ClientConfig,
  { path, schema, method = 'GET', body, signal }: RequestOptions<S>,
): Promise<z.infer<S>> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), config.timeoutMs)
  if (signal) signal.addEventListener('abort', () => controller.abort(), { once: true })

  let response: Response
  try {
    response = await fetch(`${config.baseUrl}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(config.devToken ? { Authorization: `Bearer ${config.devToken}` } : {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    })
  } catch (error) {
    clearTimeout(timeout)
    const aborted = error instanceof DOMException && error.name === 'AbortError'
    throw new AquaDocApiError({
      kind: aborted ? 'timeout' : 'network',
      code: aborted ? 'TIMEOUT' : 'NETWORK_ERROR',
      message: aborted
        ? 'The request to AquaDoc timed out.'
        : 'Could not reach AquaDoc.',
    })
  } finally {
    clearTimeout(timeout)
  }

  const requestId = response.headers.get('X-Request-ID')
  const payload: unknown = await response.json().catch(() => null)

  if (!response.ok) {
    throw toApiError(response.status, payload, requestId)
  }

  const parsed = schema.safeParse(payload)
  if (!parsed.success) {
    // Fail safe: never render a payload we could not validate.
    throw new AquaDocApiError({
      kind: 'response_invalid',
      code: 'RESPONSE_SCHEMA_MISMATCH',
      message: 'AquaDoc returned a response the frontend could not validate.',
      requestId,
      validationIssues: parsed.error.issues.map(
        (issue) => `${issue.path.join('.') || '(root)'}: ${issue.message}`,
      ),
    })
  }
  return parsed.data
}

function toApiError(status: number, payload: unknown, requestId: string | null): AquaDocApiError {
  const parsed = errorResponseSchema.safeParse(payload)

  if (parsed.success) {
    const { code, message, request_id } = parsed.data.error
    return new AquaDocApiError({
      kind: CODE_TO_KIND[code] ?? (status >= 500 ? 'server' : 'validation'),
      code,
      message,
      requestId: request_id || requestId,
    })
  }

  // A response that is not in the documented envelope — a proxy error page, or
  // a service that did not get far enough to format one.
  return new AquaDocApiError({
    kind: status >= 500 ? 'server' : status === 401 ? 'unauthorized' : 'validation',
    code: `HTTP_${status}`,
    message: `AquaDoc returned HTTP ${status}.`,
    requestId,
  })
}
