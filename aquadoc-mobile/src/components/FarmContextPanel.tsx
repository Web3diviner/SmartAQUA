/**
 * Farm Context Simulator form.
 *
 * 15_AQUADOC_FRONTEND.md section 4: before the Go backend supplies real pond
 * context, developers simulate it here.
 *
 * The critical UI rule: "The UI must render unavailable measurements as
 * `Not available`, never `0`." Every numeric input is a text field with an
 * empty default, and blank stays `null` all the way to the wire — a
 * number-typed input would coerce blank to 0 and quietly assert a measurement
 * that was never taken.
 */

import { type FarmContextForm, unmeasuredParameters } from '@/schemas/farmContext'

interface Props {
  form: FarmContextForm
  onChange: (form: FarmContextForm) => void
  onReset: () => void
  disabled?: boolean
}

export function FarmContextPanel({ form, onChange, onReset, disabled = false }: Props) {
  const update = (key: keyof FarmContextForm) => (value: string) =>
    onChange({ ...form, [key]: value })

  const unmeasured = unmeasuredParameters(form)

  return (
    <section className="farm-panel">
      <header className="farm-panel__header">
        <h2>Farm Context Simulator</h2>
        <button type="button" className="button button--ghost" onClick={onReset} disabled={disabled}>
          Reset to defaults
        </button>
      </header>

      <p className="farm-panel__note">
        Leave a field empty to mark that measurement as <strong>not taken</strong>. Empty is sent
        as <code>null</code>, which AquaDoc treats as unknown — never as zero.
      </p>

      <fieldset className="farm-panel__group" disabled={disabled}>
        <legend>Pond</legend>
        <Field label="Farm name" value={form.farm_name} onChange={update('farm_name')} />
        <Field label="Pond name" value={form.pond_name} onChange={update('pond_name')} />
        <Field label="Species" value={form.species} onChange={update('species')} />
        <Field label="Life stage" value={form.life_stage} onChange={update('life_stage')} />
        <Field
          label="Population"
          value={form.population}
          onChange={update('population')}
          hint="count"
        />
        <Field
          label="Average weight"
          value={form.average_weight_g}
          onChange={update('average_weight_g')}
          hint="g"
        />
      </fieldset>

      <fieldset className="farm-panel__group" disabled={disabled}>
        <legend>Water quality</legend>
        <Field
          label="Temperature"
          value={form.temperature_c}
          onChange={update('temperature_c')}
          hint="°C"
        />
        <Field label="pH" value={form.ph} onChange={update('ph')} hint="pH" />
        <Field
          label="Dissolved oxygen"
          value={form.dissolved_oxygen_mg_l}
          onChange={update('dissolved_oxygen_mg_l')}
          hint="mg/L"
        />
        <Field
          label="Turbidity"
          value={form.turbidity_ntu}
          onChange={update('turbidity_ntu')}
          hint="NTU"
        />
        <Field
          label="Ammonia"
          value={form.ammonia_mg_l}
          onChange={update('ammonia_mg_l')}
          hint="mg/L"
        />
        <Field
          label="Nitrite"
          value={form.nitrite_mg_l}
          onChange={update('nitrite_mg_l')}
          hint="mg/L"
        />
      </fieldset>

      <fieldset className="farm-panel__group" disabled={disabled}>
        <legend>Feeding</legend>
        <Field
          label="Daily ration"
          value={form.daily_ration_g}
          onChange={update('daily_ration_g')}
          hint="g/day"
        />
        <Field
          label="Last feeding"
          value={form.last_feeding_g}
          onChange={update('last_feeding_g')}
          hint="g"
        />
        <Field
          label="Feeds per day"
          value={form.feeds_per_day}
          onChange={update('feeds_per_day')}
        />
        <Field
          label="Feed acceptance"
          value={form.feed_acceptance}
          onChange={update('feed_acceptance')}
          hint="normal / reduced / refused"
        />
      </fieldset>

      <fieldset className="farm-panel__group" disabled={disabled}>
        <legend>Health</legend>
        <Field
          label="Mortality (24h)"
          value={form.mortality_24h}
          onChange={update('mortality_24h')}
          hint="count"
        />
        <Field
          label="Symptoms"
          value={form.reported_symptoms}
          onChange={update('reported_symptoms')}
          hint="comma-separated"
        />
      </fieldset>

      <div className="farm-panel__summary">
        <h3>Currently unmeasured</h3>
        {unmeasured.length === 0 ? (
          <p>All water parameters have values.</p>
        ) : (
          <>
            <ul className="farm-panel__unmeasured">
              {unmeasured.map((label) => (
                <li key={label}>{label}</li>
              ))}
            </ul>
            <p className="farm-panel__summary-note">
              AquaDoc will state that it could not evaluate these.
            </p>
          </>
        )}
      </div>
    </section>
  )
}

function Field({
  label,
  value,
  onChange,
  hint,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  hint?: string
}) {
  const id = `field-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`
  return (
    <div className="field">
      <label className="field__label" htmlFor={id}>
        {label}
        {hint && <span className="field__hint"> ({hint})</span>}
      </label>
      <input
        id={id}
        // Text, not number: `type="number"` would let the browser coerce, and
        // blank must survive as blank all the way to `null`.
        type="text"
        inputMode="decimal"
        className="field__input"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  )
}
