/**
 * Source citation links.
 *
 * Renders compact, interactive reference links for grounded answers without
 * bulky paragraphs or text excerpts.
 */

import type { SourceReference } from '@/schemas/aquadoc'

export function SourceCard({
  source,
  showDetails = false,
}: {
  source: SourceReference
  showDetails?: boolean
}) {
  const isUrl = source.source.startsWith('http://') || source.source.startsWith('https://')
  const displayTitle = source.title || source.source || 'Reference'
  const publisher = source.author || source.source || 'Aquaculture Guide'
  const meta = [publisher, source.year].filter(Boolean).join(', ')

  return (
    <div className="source-card">
      <a
        href={isUrl ? source.source : undefined}
        target={isUrl ? '_blank' : undefined}
        rel={isUrl ? 'noopener noreferrer' : undefined}
        className="source-link-chip"
        title={`${displayTitle}${meta ? ` (${meta})` : ''}`}
      >
        <span className="source-link-chip__title">{displayTitle}</span>
        {source.page && <span className="source-link-chip__year">{source.page}</span>}
        {!source.page && <span className="source-link-chip__year">Not paginated</span>}
      </a>
      {showDetails && (
        <div className="source-card__dev">
          <span className="source-link-chip__score">{source.score.toFixed(3)}</span>
          {source.chunk_text && (
            <button
              type="button"
              className="button button--ghost button--sm"
              onClick={() => {}}
            >
              Show full retrieved chunk
            </button>
          )}
        </div>
      )}
    </div>
  )
}

export function SourceList({
  sources,
}: {
  sources: SourceReference[]
  showDetails?: boolean
}) {
  if (!sources || sources.length === 0) {
    return null
  }

  // Deduplicate sources by title so multiple chunks from one manual don't repeat
  const uniqueSources = Array.from(
    new Map(sources.map((s) => [s.title.trim().toLowerCase(), s])).values()
  )

  return (
    <div className="sources-bar">
      <span className="sources-bar__label">
        <svg
          width="13"
          height="13"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="sources-bar__icon"
          aria-hidden="true"
        >
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
        </svg>
        Sources:
      </span>
      <div className="sources-bar__links">
        {uniqueSources.map((source, idx) => {
          const isUrl = source.source.startsWith('http://') || source.source.startsWith('https://')
          const displayTitle = source.title || source.source || `Reference ${idx + 1}`
          const publisher = source.author || source.source || 'Aquaculture Guide'
          const meta = [publisher, source.year].filter(Boolean).join(', ')

          return (
            <a
              key={`${source.document_id || 'doc'}-${idx}`}
              href={isUrl ? source.source : undefined}
              target={isUrl ? '_blank' : undefined}
              rel={isUrl ? 'noopener noreferrer' : undefined}
              className="source-link-chip"
              title={`${displayTitle}${meta ? ` (${meta})` : ''}${source.section ? ` · ${source.section}` : ''}`}
            >
              <span className="source-link-chip__title">{displayTitle}</span>
              {source.year && <span className="source-link-chip__year">{source.year}</span>}
            </a>
          )
        })}
      </div>
    </div>
  )
}
