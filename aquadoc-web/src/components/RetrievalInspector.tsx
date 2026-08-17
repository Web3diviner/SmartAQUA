/**
 * Retrieval Inspector.
 *
 * 15_AQUADOC_FRONTEND.md section 4, "Evaluation / Debug" — shows question,
 * intent, metadata filters, retrieved chunks, similarity and reranking scores,
 * evidence level, source title, page number, selected context, latency, and
 * model versions.
 *
 * The point of this screen is to make retrieval failures distinguishable from
 * generation failures (development objective 10). A weak answer over strong
 * passages is a prompt problem; a weak answer over irrelevant passages is a
 * retrieval problem. Showing rejected candidates alongside selected ones is
 * what makes that difference visible.
 *
 * Developer-facing only — never shown to farmers.
 */

import type { ChatResponse, RetrievalTrace, RetrievalTraceItem } from '@/schemas/aquadoc'

export function RetrievalInspector({ response }: { response: ChatResponse }) {
  const trace = response.retrieval_trace

  if (!trace) {
    return (
      <section className="inspector inspector--empty">
        <h3>Retrieval Inspector</h3>
        <p>
          No retrieval trace was attached to this response. Traces are developer-only and require
          the developer token.
        </p>
      </section>
    )
  }

  const selected = trace.items.filter((item) => item.selected)
  const rejected = trace.items.filter((item) => !item.selected)

  return (
    <section className="inspector">
      <h3>Retrieval Inspector</h3>

      <TraceSummary trace={trace} response={response} />

      <div className="inspector__group">
        <h4>Metadata filters</h4>
        <FilterList filters={trace.metadata_filters} />
      </div>

      <div className="inspector__group">
        <h4>Selected context ({selected.length})</h4>
        <p className="inspector__hint">These passages were sent to the model.</p>
        <CandidateTable items={selected} />
      </div>

      {rejected.length > 0 && (
        <div className="inspector__group">
          <h4>Considered but not selected ({rejected.length})</h4>
          <p className="inspector__hint">
            Retrieved and ranked, but cut by the top-K limit or the per-document diversity cap.
          </p>
          <CandidateTable items={rejected} />
        </div>
      )}

      <div className="inspector__group">
        <h4>Provenance</h4>
        <ProvenanceTable response={response} />
      </div>

      {response.confidence_breakdown && (
        <div className="inspector__group">
          <h4>Confidence inputs</h4>
          <p className="inspector__hint">
            The score blends retrieval relevance, evidence quality, data completeness, rule
            agreement, and model confidence — the model's own number is one weighted input, not
            the score.
          </p>
          <dl className="inspector__facts">
            {Object.entries(response.confidence_breakdown).map(([key, value]) => (
              <div key={key}>
                <dt>{key.replace(/_/g, ' ')}</dt>
                <dd>{value === null ? '—' : value.toFixed(4)}</dd>
              </div>
            ))}
          </dl>
        </div>
      )}
    </section>
  )
}

function TraceSummary({ trace, response }: { trace: RetrievalTrace; response: ChatResponse }) {
  return (
    <dl className="inspector__facts">
      <div>
        <dt>Request ID</dt>
        <dd><code>{trace.request_id}</code></dd>
      </div>
      <div>
        <dt>Intent</dt>
        <dd>{trace.intent}</dd>
      </div>
      <div>
        <dt>Candidates considered</dt>
        <dd>{trace.candidates_considered}</dd>
      </div>
      <div>
        <dt>Selected</dt>
        <dd>{trace.selected_count}</dd>
      </div>
      <div>
        <dt>Lexical search</dt>
        <dd>{trace.lexical_enabled ? 'Enabled (hybrid)' : 'Disabled (vector only)'}</dd>
      </div>
      <div>
        <dt>Min similarity</dt>
        <dd>{trace.min_similarity}</dd>
      </div>
      <div>
        <dt>Retrieval latency</dt>
        <dd>{trace.retrieval_latency_ms.toFixed(1)} ms</dd>
      </div>
      <div>
        <dt>Model latency</dt>
        <dd>{response.provenance.llm_latency_ms.toFixed(1)} ms</dd>
      </div>
      <div>
        <dt>Total latency</dt>
        <dd>{response.provenance.total_latency_ms.toFixed(1)} ms</dd>
      </div>
      <div>
        <dt>Tokens (in / out)</dt>
        <dd>
          {response.provenance.input_tokens ?? '—'} / {response.provenance.output_tokens ?? '—'}
        </dd>
      </div>
    </dl>
  )
}

function FilterList({ filters }: { filters: Record<string, string[]> }) {
  const entries = Object.entries(filters)
  if (entries.length === 0) return <p className="inspector__hint">No metadata filters applied.</p>

  return (
    <ul className="inspector__filters">
      {entries.map(([key, values]) => (
        <li key={key}>
          <code>{key}</code>: {values.join(', ')}
        </li>
      ))}
    </ul>
  )
}

function CandidateTable({ items }: { items: RetrievalTraceItem[] }) {
  if (items.length === 0) return <p className="inspector__hint">No passages.</p>

  return (
    <div className="table-scroll">
      <table className="inspector__table">
        <thead>
          <tr>
            <th>Document</th>
            <th>Page</th>
            <th>Ev.</th>
            <th>Similarity</th>
            <th>Vec rank</th>
            <th>Lex rank</th>
            <th>Fused</th>
            <th>Final</th>
            <th>Preview</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.chunk_id}>
              <td>
                {item.title}
                {item.section && <span className="inspector__section"> · {item.section}</span>}
              </td>
              <td>{item.page ?? '—'}</td>
              <td>{item.evidence_level}</td>
              <td>{item.similarity.toFixed(4)}</td>
              <td>{item.vector_rank ?? '—'}</td>
              {/* An em dash means the lexical search did not match this chunk. */}
              <td>{item.lexical_rank ?? '—'}</td>
              <td>{item.fused_score.toFixed(5)}</td>
              <td>{item.final_score.toFixed(5)}</td>
              <td className="inspector__preview">{item.content_preview}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ProvenanceTable({ response }: { response: ChatResponse }) {
  const { provenance } = response
  return (
    <dl className="inspector__facts">
      <div>
        <dt>Prompt version</dt>
        <dd><code>{provenance.prompt_version}</code></dd>
      </div>
      <div>
        <dt>LLM</dt>
        <dd>
          {provenance.llm_provider} / <code>{provenance.llm_model}</code>
        </dd>
      </div>
      <div>
        <dt>Embeddings</dt>
        <dd>
          {provenance.embedding_provider} / <code>{provenance.embedding_model}</code>
        </dd>
      </div>
      <div>
        <dt>Rules version</dt>
        <dd><code>{provenance.rules_version}</code></dd>
      </div>
      <div>
        <dt>Farm context</dt>
        <dd>
          {provenance.farm_context_supplied
            ? `Supplied (${(provenance.farm_context_completeness * 100).toFixed(0)}% complete)`
            : 'Not supplied'}
        </dd>
      </div>
      <div>
        <dt>Generated at</dt>
        <dd>{provenance.generated_at}</dd>
      </div>
    </dl>
  )
}
