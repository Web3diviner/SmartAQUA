/**
 * Missing-data display.
 *
 * 15_AQUADOC_FRONTEND.md section 14: for contextual questions, clearly show
 * what AquaDoc could not evaluate.
 *
 *   Data not currently available
 *   - pH
 *   - Dissolved Oxygen
 *   - Turbidity
 *
 * This matters because temperature is currently the only active water-quality
 * measurement; the other sensors are not installed yet. Rendering an
 * unavailable parameter as `0` would misrepresent a pond as measured-and-fine.
 */

interface Props {
  /** Human-readable labels, e.g. "pH", "Dissolved Oxygen". */
  labels: string[]
  /** Suppressed for general questions, where farm data is not expected. */
  hidden?: boolean
}

export function MissingDataPanel({ labels, hidden = false }: Props) {
  if (hidden) return null

  if (labels.length === 0) {
    return (
      <div className="missing-data missing-data--none">
        <span className="missing-data__title">Missing Data</span>
        <span className="missing-data__value">None — all tracked measurements are available.</span>
      </div>
    )
  }

  return (
    <div className="missing-data">
      <span className="missing-data__title">Data not currently available</span>
      <ul className="missing-data__list">
        {labels.map((label) => (
          <li key={label}>{label}</li>
        ))}
      </ul>
      <p className="missing-data__note">
        These measurements were not taken, so AquaDoc could not evaluate them. They are unknown —
        not normal.
      </p>
    </div>
  )
}

/**
 * Render a single measurement value.
 *
 * The one place a measurement becomes display text. `null` renders as
 * "Not available", never as `0` — a truthiness check here would also hide a
 * genuine 0.0 mg/L dissolved-oxygen reading, which is a pond-killing event.
 */
export function MeasurementValue({ value, unit }: { value: number | null; unit?: string }) {
  if (value === null || value === undefined) {
    return <span className="measurement measurement--unavailable">Not available</span>
  }
  return (
    <span className="measurement">
      {value}
      {unit ? <span className="measurement__unit"> {unit}</span> : null}
    </span>
  )
}
