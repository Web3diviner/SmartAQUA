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
  model?: string | null
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
      model: params.model || undefined,
    },
  })
}

/** Transcribe audio recording via Groq Whisper. */
export async function transcribeAudio(
  config: ClientConfig,
  audioBlob: Blob,
  model = 'whisper-large-v3-turbo',
): Promise<{ text: string }> {
  const url = `${config.baseUrl.replace(/\/$/, '')}/dev/v1/audio/transcribe`
  const formData = new FormData()
  formData.append('file', audioBlob, 'recording.wav')
  formData.append('model', model)

  const headers: Record<string, string> = {}
  if (config.devToken) {
    headers.Authorization = `Bearer ${config.devToken}`
  }

  const response = await fetch(url, {
    method: 'POST',
    headers,
    body: formData,
  })

  if (!response.ok) {
    throw new Error(`Audio transcription failed (${response.status})`)
  }

  return (await response.json()) as { text: string }
}

/** Restore punctuation and casing via backend AI model if available. */
export async function punctuateText(config: ClientConfig, text: string): Promise<string> {
  if (!text || !text.trim()) return ''
  try {
    const url = `${config.baseUrl.replace(/\/$/, '')}/dev/v1/text/punctuate`
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (config.devToken) {
      headers.Authorization = `Bearer ${config.devToken}`
    }
    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify({ text }),
    })
    if (response.ok) {
      const data = (await response.json()) as { text: string }
      return data.text || text
    }
  } catch {
    // Fallback if backend is offline
  }
  return text
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

export async function deleteConversationApi(config: ClientConfig, conversationId: string): Promise<boolean> {
  try {
    const url = `${config.baseUrl.replace(/\/$/, '')}/dev/v1/conversations/${encodeURIComponent(conversationId)}`
    const headers: Record<string, string> = {}
    if (config.devToken) {
      headers.Authorization = `Bearer ${config.devToken}`
    }
    const res = await fetch(url, { method: 'DELETE', headers })
    return res.ok
  } catch {
    return false
  }
}
