/**
 * Groq AI Model Catalog & Metadata.
 *
 * Supported models hosted on https://console.groq.com/
 */

export interface ModelOption {
  id: string
  name: string
  category: 'reasoning' | 'safeguard' | 'compound' | 'audio'
  badge: string
  description: string
  isDefault?: boolean
}

export const GROQ_MODELS: ModelOption[] = [
  // Deep Reasoning & LLMs
  {
    id: 'openai/gpt-oss-120b',
    name: 'GPT-OSS 120B',
    category: 'reasoning',
    badge: 'Flagship Reasoning',
    description: 'High-capacity reasoning, complex disease triage, and feeding calculations.',
    isDefault: true,
  },
  {
    id: 'openai/gpt-oss-20b',
    name: 'GPT-OSS 20B',
    category: 'reasoning',
    badge: 'Fast Reasoning',
    description: 'Fast, balanced reasoning optimized for rapid aquaculture Q&A.',
  },
  {
    id: 'qwen/qwen3.8-27b',
    name: 'Qwen 3.8 27B',
    category: 'reasoning',
    badge: 'Multilingual RAG',
    description: 'Domain-specific aquaculture knowledge and multilingual reasoning.',
  },
  {
    id: 'groq/compound-mini',
    name: 'Compound Mini',
    category: 'compound',
    badge: 'Compound Agent',
    description: 'Ultra-low latency compound agentic reasoning pipeline.',
  },

  // Safeguard & Moderation
  {
    id: 'openai/gpt-oss-safeguard-20b',
    name: 'GPT-OSS Safeguard 20B',
    category: 'safeguard',
    badge: 'Safety Boundary',
    description: 'Deterministic safety boundary validation and hallucination mitigation.',
  },
  {
    id: 'meta-llama/llama-prompt-guard-2-86m',
    name: 'Prompt Guard 2 (86M)',
    category: 'safeguard',
    badge: 'Input Shield',
    description: 'Deep input safety inspection detecting prompt injections and adversarial inputs.',
  },
  {
    id: 'meta-llama/llama-prompt-guard-2-22m',
    name: 'Prompt Guard 2 (22M)',
    category: 'safeguard',
    badge: 'Fast Guard',
    description: 'Ultra-lightweight prompt security and content safety filter.',
  },

  // Audio / Speech-to-Text
  {
    id: 'whisper-large-v3',
    name: 'Whisper Large v3',
    category: 'audio',
    badge: 'Audio STT',
    description: 'High-accuracy multilingual speech recognition for voice notes.',
  },
  {
    id: 'whisper-large-v3-turbo',
    name: 'Whisper Large v3 Turbo',
    category: 'audio',
    badge: 'Ultra-fast STT',
    description: 'Real-time speech-to-text transcription for voice queries.',
  },
]

export const DEFAULT_MODEL_ID = 'openai/gpt-oss-120b'
