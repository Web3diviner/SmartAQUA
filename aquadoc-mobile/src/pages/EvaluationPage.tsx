/**
 * AI Evaluation & Debug Dashboard Page.
 *
 * 15_AQUADOC_FRONTEND.md section 4 & section 7: Developer-facing analytics
 * for inspecting RAG latency, LLM model version, reranking score distribution,
 * token consumption, and deterministic safety rule execution logs.
 */

import { useState } from 'react'

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
}

const MOCK_TRACES: TraceMetric[] = [
  {
    id: 'REQ-9014',
    question: 'What causes surface piping in catfish ponds during early morning hours?',
    intent: 'water_quality_triage',
    retrieval_ms: 112,
    llm_ms: 640,
    total_ms: 752,
    total_tokens: 1240,
    cost_usd: 0.0037,
    confidence: 0.88,
    rule_pass_rate: '14 / 14',
    created_at: '2026-08-27 16:40:12',
  },
  {
    id: 'REQ-9015',
    question: 'Calculate daily feed ration for 500kg biomass at 28.5 deg C.',
    intent: 'feeding_calculation',
    retrieval_ms: 85,
    llm_ms: 420,
    total_ms: 505,
    total_tokens: 890,
    cost_usd: 0.0026,
    confidence: 0.94,
    rule_pass_rate: '14 / 14',
    created_at: '2026-08-27 16:42:05',
  },
  {
    id: 'REQ-9016',
    question: 'Is 0.5 mg/L dissolved oxygen safe for fingerlings?',
    intent: 'safety_boundary',
    retrieval_ms: 98,
    llm_ms: 510,
    total_ms: 608,
    total_tokens: 1020,
    cost_usd: 0.0030,
    confidence: 0.96,
    rule_pass_rate: '14 / 14',
    created_at: '2026-08-27 16:45:30',
  },
]

export function EvaluationPage() {
  const [selectedTrace, setSelectedTrace] = useState<TraceMetric>(MOCK_TRACES[0]!)

  return (
    <div className="eval-page">
      <header className="page-header">
        <div>
          <span className="page-eyebrow">Performance & AI Telemetry Console</span>
          <h2>AquaDoc Evaluation Hub</h2>
        </div>
        <p>
          Inspect token consumption, vector retrieval latency, grounding accuracy, and safety rule passes.
        </p>
      </header>

      <div className="eval-metrics-grid">
        <div className="eval-card">
          <span className="eval-card__label">Avg Total Latency</span>
          <div className="eval-card__value">621 <small>ms</small></div>
          <span className="eval-card__sub text-ok">&darr; 14% vs baseline</span>
        </div>

        <div className="eval-card">
          <span className="eval-card__label">Avg RAG Accuracy</span>
          <div className="eval-card__value">92.4 <small>%</small></div>
          <span className="eval-card__sub text-ok">High grounding grade</span>
        </div>

        <div className="eval-card">
          <span className="eval-card__label">Safety Rule Pass Rate</span>
          <div className="eval-card__value">100 <small>%</small></div>
          <span className="eval-card__sub">0 safety violations</span>
        </div>

        <div className="eval-card">
          <span className="eval-card__label">Est. Cost / 1k Queries</span>
          <div className="eval-card__value">$3.10</div>
          <span className="eval-card__sub">Production telemetry</span>
        </div>
      </div>

      <div className="eval-content">
        <div className="trace-table-container">
          <div className="table-header">
            <h3>Recent Query Execution Traces</h3>
            <span>Showing latest {MOCK_TRACES.length} requests</span>
          </div>

          <table className="trace-table">
            <thead>
              <tr>
                <th>Request ID</th>
                <th>Intent</th>
                <th>Question</th>
                <th>Confidence</th>
                <th>Total Latency</th>
                <th>Tokens</th>
              </tr>
            </thead>
            <tbody>
              {MOCK_TRACES.map((trace) => (
                <tr
                  key={trace.id}
                  className={selectedTrace.id === trace.id ? 'row--selected' : ''}
                  onClick={() => setSelectedTrace(trace)}
                >
                  <td className="font-mono">{trace.id}</td>
                  <td>
                    <span className="intent-badge">{trace.intent}</span>
                  </td>
                  <td className="cell-question">{trace.question}</td>
                  <td>
                    <span className="confidence-pill">
                      {(trace.confidence * 100).toFixed(0)}%
                    </span>
                  </td>
                  <td className="font-mono">{trace.total_ms} ms</td>
                  <td className="font-mono">{trace.total_tokens}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <aside className="trace-detail-panel">
          <div className="panel-header">
            <span className="badge badge--primary">{selectedTrace.id}</span>
            <span className="timestamp">{selectedTrace.created_at}</span>
          </div>

          <h3>Execution Details</h3>

          <div className="detail-group">
            <h4>Question & Intent</h4>
            <p className="question-box">{selectedTrace.question}</p>
            <div className="meta-pair">
              <span>Intent Classification</span>
              <strong>{selectedTrace.intent}</strong>
            </div>
          </div>

          <div className="detail-group">
            <h4>Latency & Token Metrics</h4>
            <div className="latency-bar">
              <div
                className="latency-seg latency-seg--rag"
                style={{ width: `${(selectedTrace.retrieval_ms / selectedTrace.total_ms) * 100}%` }}
                title={`Retrieval: ${selectedTrace.retrieval_ms}ms`}
              />
              <div
                className="latency-seg latency-seg--llm"
                style={{ width: `${(selectedTrace.llm_ms / selectedTrace.total_ms) * 100}%` }}
                title={`LLM Generation: ${selectedTrace.llm_ms}ms`}
              />
            </div>
            <div className="latency-legend">
              <span><i className="dot dot--rag" /> Vector RAG ({selectedTrace.retrieval_ms}ms)</span>
              <span><i className="dot dot--llm" /> LLM Gen ({selectedTrace.llm_ms}ms)</span>
            </div>

            <dl className="kv-grid mt-2">
              <div>
                <dt>Total Tokens</dt>
                <dd>{selectedTrace.total_tokens} tokens</dd>
              </div>
              <div>
                <dt>Est. Query Cost</dt>
                <dd>${selectedTrace.cost_usd.toFixed(4)}</dd>
              </div>
            </dl>
          </div>

          <div className="detail-group">
            <h4>Deterministic Rule Validations</h4>
            <div className="rule-box text-ok">
              <span>✓ Safety Checks Passed ({selectedTrace.rule_pass_rate})</span>
              <p>No critical parameter bounds broken. Missing parameter labels correctly generated.</p>
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}
