/**
 * Risk level display.
 *
 * Risk describes the situation; a recommendation's tier describes what approval
 * an action needs. They are deliberately separate concepts
 * (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 5).
 */

import type { RiskLevel } from '@/schemas/aquadoc'

const LABELS: Record<RiskLevel, string> = {
  informational: 'Informational',
  watch: 'Watch',
  elevated: 'Elevated',
  high: 'High',
}

export function RiskBadge({ level }: { level: RiskLevel }) {
  return (
    <span className={`badge badge--risk badge--risk-${level}`}>
      <span className="badge__label">Risk</span>
      <span className="badge__value">{LABELS[level]}</span>
    </span>
  )
}
