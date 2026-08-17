/** Chat and diagnostics endpoints. */

import { type ClientConfig, request } from '@/api/client'
import {
  type ChatResponse,
  type ConfigResponse,
  type RetrievalTrace,
  chatResponseSchema,
  configResponseSchema,
  retrievalTraceSchema,
} from '@/schemas/aquadoc'
import type { FarmContext } from '@/schemas/farmContext'

export interface ChatParams {
  userId: string
  question: string
  conversationId: string | null
  /** Omitted entirely for General Aquaculture mode. */
  farmContext: FarmContext | null
  signal?: AbortSignal
}

/**
 * Ask AquaDoc a question.
 *
 * Targets `/dev/v1/chat`, which runs the identical orchestrator as the internal
 * endpoint but attaches the retrieval trace and confidence breakdown. The
 * request and response shapes are the same either way, so switching this client
 * to go through the Go backend later is a URL change
 * (15_AQUADOC_FRONTEND.md section 19).
 */
export async function sendChat(config: ClientConfig, params: ChatParams): Promise<ChatResponse> {
  return request(config, {
    path: '/dev/v1/chat',
    method: 'POST',
    schema: chatResponseSchema,
    signal: params.signal,
    body: {
      user_id: params.userId,
      question: params.question,
      conversation_id: params.conversationId,
      // `null` rather than an empty object: an empty context would look like a
      // pond where every measurement is unknown, which is a different claim
      // from "this is a general question".
      farm_context: params.farmContext,
    },
  })
}

export async function fetchRetrievalTrace(
  config: ClientConfig,
  requestId: string,
): Promise<RetrievalTrace> {
  return request(config, {
    path: `/dev/v1/debug/retrieval/${encodeURIComponent(requestId)}`,
    schema: retrievalTraceSchema,
  })
}

export async function fetchConfig(config: ClientConfig): Promise<ConfigResponse> {
  return request(config, { path: '/dev/v1/config', schema: configResponseSchema })
}
