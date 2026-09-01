/**
 * Runtime validation of every AquaDoc response.
 *
 * 15_AQUADOC_FRONTEND.md section 11: validate API responses; invalid backend
 * responses should fail safely.
 *
 * These schemas mirror `aquadoc/app/schemas/chat.py`. They are the frontend's
 * half of the contract that 15_AQUADOC_FRONTEND.md section 19 says must stay
 * stable through the Flutter migration — so when a field is added here, the
 * corresponding Pydantic model changes too.
 *
 * Optional fields use `.nullish()` rather than `.optional()` because the
 * service omits nulls on the wire but Python may serialise an explicit `null`.
 * Both must parse.
 */

import { z } from 'zod'

export const intentSchema = z.enum([
  'general_aquaculture',
  'farm_assessment',
  'feeding',
  'water_quality',
  'disease',
  'unknown',
])

export const riskLevelSchema = z.enum(['informational', 'watch', 'elevated', 'high'])

export const confidenceBandSchema = z.enum(['low', 'moderate', 'high'])

export const evidenceLevelSchema = z.enum(['A', 'B', 'C', 'D', 'E'])

export const recommendationTierSchema = z.enum([
  'tier_0_informational',
  'tier_1_advisory',
  'tier_2_low_risk_operational',
  'tier_3_high_risk',
])

export const sourceReferenceSchema = z.object({
  chunk_id: z.string(),
  document_id: z.string(),
  title: z.string(),
  source: z.string(),
  author: z.string().nullish(),
  year: z.number().int().nullish(),
  // Nullable: not every document type is paginated.
  page: z.number().int().nullish(),
  section: z.string().nullish(),
  evidence_level: evidenceLevelSchema,
  excerpt: z.string(),
  score: z.number().min(0).max(1),
  // Developer mode only.
  chunk_text: z.string().nullish(),
})

export const possibleCauseSchema = z.object({
  name: z.string(),
  confidence: z.number().min(0).max(1),
  explanation: z.string().nullish(),
  supporting_source_ids: z.array(z.string()).default([]),
})

export const recommendedActionSchema = z.object({
  action: z.string(),
  tier: recommendationTierSchema,
  reason: z.string(),
  requires_approval: z.boolean(),
  urgency: riskLevelSchema.default('informational'),
})

export const ruleFindingSchema = z.object({
  rule_id: z.string(),
  rule_version: z.string(),
  status: z.string(),
  summary: z.string(),
  measurement: z.string().nullish(),
  observed: z.number().nullish(),
  expected_range: z.tuple([z.number(), z.number()]).nullish(),
})

export const retrievalTraceItemSchema = z.object({
  chunk_id: z.string(),
  document_id: z.string(),
  title: z.string(),
  page: z.number().int().nullish(),
  section: z.string().nullish(),
  evidence_level: evidenceLevelSchema,
  similarity: z.number(),
  lexical_rank: z.number().int().nullish(),
  vector_rank: z.number().int().nullish(),
  fused_score: z.number(),
  final_score: z.number(),
  selected: z.boolean(),
  content_preview: z.string(),
})

export const retrievalTraceSchema = z.object({
  request_id: z.string(),
  question: z.string(),
  intent: intentSchema,
  metadata_filters: z.record(z.array(z.string())).default({}),
  embedding_model: z.string(),
  embedding_dimensions: z.number().int(),
  candidates_considered: z.number().int(),
  selected_count: z.number().int(),
  lexical_enabled: z.boolean(),
  min_similarity: z.number(),
  items: z.array(retrievalTraceItemSchema).default([]),
  retrieval_latency_ms: z.number().default(0),
})

export const provenanceSchema = z.object({
  prompt_version: z.string(),
  llm_model: z.string(),
  llm_provider: z.string(),
  embedding_model: z.string(),
  embedding_provider: z.string(),
  rules_version: z.string(),
  retrieval_source_ids: z.array(z.string()).default([]),
  farm_context_supplied: z.boolean().default(false),
  farm_context_completeness: z.number().default(0),
  generated_at: z.string(),
  llm_latency_ms: z.number().default(0),
  total_latency_ms: z.number().default(0),
  input_tokens: z.number().int().nullish(),
  output_tokens: z.number().int().nullish(),
})

export const chatResponseSchema = z.object({
  request_id: z.string(),
  conversation_id: z.string(),
  answer: z.string(),
  intent: intentSchema,
  risk_level: riskLevelSchema,
  confidence: z.number().min(0).max(1),
  confidence_band: confidenceBandSchema,
  possible_causes: z.array(possibleCauseSchema).default([]),
  recommended_actions: z.array(recommendedActionSchema).default([]),
  /** Measurement keys AquaDoc could not evaluate. Rendered as "Not available". */
  missing_data: z.array(z.string()).default([]),
  missing_data_labels: z.array(z.string()).default([]),
  expert_escalation: z.boolean().default(false),
  escalation_reasons: z.array(z.string()).default([]),
  sources: z.array(sourceReferenceSchema).default([]),
  rule_findings: z.array(ruleFindingSchema).default([]),
  warnings: z.array(z.string()).default([]),
  provenance: provenanceSchema,
  retrieval_trace: retrievalTraceSchema.nullish(),
  confidence_breakdown: z.record(z.number().nullable()).nullish(),
})

/** 05_API_AND_SERVICE_CONTRACTS.md, "Error Format". */
export const errorResponseSchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
    request_id: z.string(),
    details: z.record(z.unknown()).default({}),
  }),
})

export const configResponseSchema = z.object({
  environment: z.string(),
  llm_provider: z.string(),
  llm_model: z.string(),
  llm_effort: z.string(),
  embedding_provider: z.string(),
  embedding_model: z.string(),
  embedding_dimensions: z.number().int(),
  retrieval_candidates: z.number().int(),
  retrieval_top_k: z.number().int(),
  retrieval_min_similarity: z.number(),
  retrieval_lexical_enabled: z.boolean(),
  chunk_target_tokens: z.number().int(),
  chunk_overlap_tokens: z.number().int(),
  rules_version: z.string(),
  prompt_versions: z.record(z.string()).default({}),
  water_quality_parameters: z.array(z.string()).default([]),
})

export type Intent = z.infer<typeof intentSchema>
export type RiskLevel = z.infer<typeof riskLevelSchema>
export type ConfidenceBand = z.infer<typeof confidenceBandSchema>
export type EvidenceLevel = z.infer<typeof evidenceLevelSchema>
export type RecommendationTier = z.infer<typeof recommendationTierSchema>
export type SourceReference = z.infer<typeof sourceReferenceSchema>
export type PossibleCause = z.infer<typeof possibleCauseSchema>
export type RecommendedAction = z.infer<typeof recommendedActionSchema>
export type RuleFinding = z.infer<typeof ruleFindingSchema>
export type RetrievalTrace = z.infer<typeof retrievalTraceSchema>
export type RetrievalTraceItem = z.infer<typeof retrievalTraceItemSchema>
export type Provenance = z.infer<typeof provenanceSchema>
export type ChatResponse = z.infer<typeof chatResponseSchema>
export type ConfigResponse = z.infer<typeof configResponseSchema>
