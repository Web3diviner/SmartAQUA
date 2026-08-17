/**
 * Farm Simulator page.
 *
 * Hosts the context form plus a live preview of the exact JSON that will be
 * sent with the next question. The preview exists because the missing-data
 * contract is easiest to verify by eye: `"ph": null` in the payload is the
 * difference between "unknown" and "measured as zero".
 */

import { useAppState } from '@/app/providers'
import { FarmContextPanel } from '@/components/FarmContextPanel'
import { MeasurementValue } from '@/components/MissingDataPanel'
import { formToFarmContext } from '@/schemas/farmContext'

export function FarmSimulatorPage() {
  const { farmForm, setFarmForm, resetFarmForm, chatMode, setChatMode } = useAppState()
  const context = formToFarmContext(farmForm)

  const biomassKg =
    context.population !== null && context.average_weight_g !== null
      ? (context.population * context.average_weight_g) / 1000
      : null

  return (
    <div className="simulator-page">
      <header className="simulator-page__heading">
        <div>
          <span className="page-eyebrow">Farm context</span>
          <h2>Give AquaDoc the full pond picture.</h2>
        </div>
        <p>Simulate real farm conditions and see exactly what AquaDoc uses to shape its guidance.</p>
      </header>
      <FarmContextPanel form={farmForm} onChange={setFarmForm} onReset={resetFarmForm} />

      <aside className="simulator-page__preview">
        <span className="page-eyebrow">Live interpretation</span>
        <h2>Computed pond state</h2>

        {chatMode !== 'simulated_pond' && (
          <div className="notice notice--warning">
            <p>
              Chat is currently in <strong>General Aquaculture</strong> mode, so this context is
              not sent with questions.
            </p>
            <button
              type="button"
              className="button"
              onClick={() => setChatMode('simulated_pond')}
            >
              Switch chat to Simulated Pond
            </button>
          </div>
        )}

        <dl className="simulator-page__facts">
          <div>
            <dt>Biomass</dt>
            <dd>
              <MeasurementValue value={biomassKg === null ? null : Number(biomassKg.toFixed(2))} unit="kg" />
            </dd>
          </div>
          <div>
            <dt>Temperature</dt>
            <dd><MeasurementValue value={context.water.temperature_c} unit="°C" /></dd>
          </div>
          <div>
            <dt>pH</dt>
            <dd><MeasurementValue value={context.water.ph} /></dd>
          </div>
          <div>
            <dt>Dissolved oxygen</dt>
            <dd><MeasurementValue value={context.water.dissolved_oxygen_mg_l} unit="mg/L" /></dd>
          </div>
          <div>
            <dt>Turbidity</dt>
            <dd><MeasurementValue value={context.water.turbidity_ntu} unit="NTU" /></dd>
          </div>
          <div>
            <dt>Daily ration</dt>
            <dd><MeasurementValue value={context.feeding.daily_ration_g} unit="g" /></dd>
          </div>
          <div>
            <dt>Mortality (24h)</dt>
            <dd><MeasurementValue value={context.health.mortality_24h} /></dd>
          </div>
        </dl>

        <h3>Payload sent to AquaDoc</h3>
        <p className="simulator-page__hint">
          <code>null</code> means the measurement was not taken. AquaDoc reports these as
          unevaluated rather than assuming a value.
        </p>
        <pre className="simulator-page__json">{JSON.stringify(context, null, 2)}</pre>
      </aside>
    </div>
  )
}
