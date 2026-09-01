/**
 * Confidence display.
 *
 * 15_AQUADOC_FRONTEND.md section 13: "Do not imply false scientific precision
 * to farmers." Farmers see Low / Moderate / High; the numeric value is
 * developer-facing only.
 *
 * The band comes from the service rather than being recomputed here, so the
 * boundaries live in one place and cannot drift between client and server.
 */

import type { ConfidenceBand } from '@/schemas/aquadoc'

const LABELS: Record<ConfidenceBand, string> = {
  low: 'Low',
  moderate: 'Moderate',
  high: 'High',
}

interface Props {
  band: ConfidenceBand
  /** Numeric score. Rendered only when `showNumeric` is set. */
  score: number
  showNumeric?: boolean
}

export function ConfidenceBadge({ band, score, showNumeric = false }: Props) {
  return (
    <span className={`badge badge--confidence badge--${band}`}>
      <span className="badge__label">Confidence</span>
      <span className="badge__value">
        {LABELS[band]}
        {showNumeric && <span className="badge__numeric"> ({score.toFixed(2)})</span>}
      </span>
    </span>
  )
}
