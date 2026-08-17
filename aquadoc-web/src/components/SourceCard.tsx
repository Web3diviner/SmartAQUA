/**
 * Source citation card.
 *
 * 15_AQUADOC_FRONTEND.md section 12: every grounded answer displays supporting
 * sources, with title and page. Developer mode additionally shows the actual
 * retrieved chunk and its retrieval score.
 */

import { useState } from 'react'
import type { SourceReference } from '@/schemas/aquadoc'

/** 04_AQUADOC_RAG_LLM.md section 5. */
const EVIDENCE_LABELS: Record<string, string> = {
  A: 'A — Official / expert-reviewed guideline',
  B: 'B — Peer-reviewed research',
  C: 'C — Established textbook or manual',
  D: 'D — Verified Smart Aqua expert case',
  E: 'E — Farmer / user report',
}

export function SourceCard({
  source,
  showDetails = false,
}: {
  source: SourceReference
  showDetails?: boolean
}) {
  const [expanded, setExpanded] = useState(false)

  return (
    <article className="source-card">
      <header className="source-card__header">
        <span className="source-card__id">{source.chunk_id}</span>
        <div className="source-card__titles">
          <h4 className="source-card__title">{source.title}</h4>
          <p className="source-card__meta">
            {source.source}
            {source.author ? ` · ${source.author}` : ''}
            {source.year ? ` · ${source.year}` : ''}
          </p>
        </div>
      </header>

      <dl className="source-card__facts">
        <div>
          <dt>Page</dt>
          {/* Nullable: not every document type is paginated. */}
          <dd>{source.page ?? 'Not paginated'}</dd>
        </div>
        {source.section && (
          <div>
            <dt>Section</dt>
            <dd>{source.section}</dd>
          </div>
        )}
        <div>
          <dt>Evidence</dt>
          <dd title={EVIDENCE_LABELS[source.evidence_level]}>{source.evidence_level}</dd>
        </div>
        {showDetails && (
          <div>
            <dt>Similarity</dt>
            <dd>{source.score.toFixed(3)}</dd>
          </div>
        )}
      </dl>

      <blockquote className="source-card__excerpt">{source.excerpt}</blockquote>

      {/* Developer mode: the exact text the model was given. */}
      {showDetails && source.chunk_text && (
        <div className="source-card__chunk">
          <button
            type="button"
            className="button button--ghost"
            onClick={() => setExpanded((value) => !value)}
            aria-expanded={expanded}
          >
            {expanded ? 'Hide' : 'Show'} full retrieved chunk
          </button>
          {expanded && <pre className="source-card__chunk-text">{source.chunk_text}</pre>}
        </div>
      )}
    </article>
  )
}

export function SourceList({
  sources,
  showDetails = false,
}: {
  sources: SourceReference[]
  showDetails?: boolean
}) {
  if (sources.length === 0) {
    return (
      <div className="sources sources--empty">
        <h3 className="sources__heading">Sources</h3>
        <p className="sources__empty-note">
          No approved knowledge sources were retrieved for this question. The answer is not
          grounded in the knowledge base, and its confidence is capped accordingly.
        </p>
      </div>
    )
  }

  return (
    <div className="sources">
      <h3 className="sources__heading">Sources ({sources.length})</h3>
      <div className="sources__list">
        {sources.map((source) => (
          <SourceCard key={source.chunk_id} source={source} showDetails={showDetails} />
        ))}
      </div>
    </div>
  )
}
