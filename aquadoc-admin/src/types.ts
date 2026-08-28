export interface Booking {
  id: string
  farmer_name: string
  farmer_phone: string
  farm_location: string
  booking_type: 'physical' | 'virtual'
  species: string
  symptoms: string[]
  preferred_date: string
  notes: string
  status: 'pending' | 'confirmed' | 'dispatched' | 'completed' | 'cancelled'
  assigned_vet: string | null
  created_at: string
}

export interface DailyUserMetric {
  date: string
  active_users: number
  new_onboarded: number
}

export interface RegionalMetric {
  region: string
  count: number
  percentage: number
}

export interface ConditionMetric {
  condition: string
  cases: number
  severity: 'critical' | 'high' | 'moderate' | 'low'
}

export interface AnalyticsKPIs {
  total_users_onboarded: number
  onboarded_growth_mom_pct: number
  daily_active_users: number
  dau_growth_wow_pct: number
  total_ponds_monitored: number
  total_triage_sessions: number
  pending_bookings_count: number
  total_bookings_count: number
}

export interface SystemBenchmarks {
  rag_grounding_accuracy_pct: number
  avg_retrieval_latency_ms: number
  avg_llm_latency_ms: number
  daily_tokens_processed: number
  error_rate_pct: number
}

export interface AdminAnalyticsResponse {
  kpis: AnalyticsKPIs
  daily_users_trend: DailyUserMetric[]
  regional_distribution: RegionalMetric[]
  top_diagnosed_conditions: ConditionMetric[]
  system_benchmarks: SystemBenchmarks
}

export interface TraceMetric {
  id: string
  question: string
  intent: string
  retrieval_ms: number
  llm_ms: number
  total_ms: number
  total_tokens: number
  cost_usd: number
  confidence: number
  rule_pass_rate: string
  created_at: string
  model?: string
}
