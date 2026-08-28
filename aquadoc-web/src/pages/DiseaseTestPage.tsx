/**
 * Disease Assessment Tester Page.
 *
 * 15_AQUADOC_FRONTEND.md section 4: Guided disease triage tool for fish health issues.
 * Provides structured symptom inputs and generates differential diagnostic
 * evaluations, confidence indicators, and safety escalation paths.
 */

import { useState } from 'react'

export interface DiagnosticResult {
  condition: string
  confidence: number
  risk_level: 'low' | 'moderate' | 'high' | 'critical'
  supporting_evidence: string[]
  missing_data: string[]
  recommended_steps: string[]
  expert_escalation: boolean
}

const COMMON_SYMPTOMS = [
  'Loss of appetite / Refusal to feed',
  'Surface piping / Gasping at surface',
  'Lethargy & slow swimming',
  'Skin lesions, ulcers, or white patches',
  'Pale or congested gills',
  'Erratic swimming / Whirling',
  'Abdominal distension (Dropsy)',
  'Fraying or erosion of fins',
]

export function DiseaseTestPage() {
  const [species, setSpecies] = useState('Clarias gariepinus')
  const [stage, setStage] = useState('Grow-out')
  const [durationDays, setDurationDays] = useState<number>(2)
  const [mortality24h, setMortality24h] = useState<number>(5)
  const [selectedSymptoms, setSelectedSymptoms] = useState<string[]>([
    'Surface piping / Gasping at surface',
    'Loss of appetite / Refusal to feed',
  ])
  const [waterTemp, setWaterTemp] = useState<string>('29.5')
  const [dissolvedOxygen, setDissolvedOxygen] = useState<string>('')

  const [result, setResult] = useState<DiagnosticResult | null>(null)
  const [evaluating, setEvaluating] = useState(false)

  const toggleSymptom = (symptom: string) => {
    setSelectedSymptoms((prev) =>
      prev.includes(symptom) ? prev.filter((s) => s !== symptom) : [...prev, symptom],
    )
  }

  const handleAssess = (e: React.FormEvent) => {
    e.preventDefault()
    setEvaluating(true)

    setTimeout(() => {
      const isOxygenDepletion = selectedSymptoms.includes('Surface piping / Gasping at surface')
      const hasLesions = selectedSymptoms.includes('Skin lesions, ulcers, or white patches')

      if (isOxygenDepletion) {
        setResult({
          condition: 'Acute Anoxia / Dissolved Oxygen Depletion',
          confidence: 0.88,
          risk_level: 'critical',
          supporting_evidence: [
            'Surface piping behavior under high morning temperature',
            'Refusal to feed during feeding hour',
            'Elevated 24h mortality rate',
          ],
          missing_data: dissolvedOxygen === '' ? ['Dissolved Oxygen Measurement (mg/L)', 'Ammonia Level'] : [],
          recommended_steps: [
            'Immediately activate emergency aeration / paddlewheel system',
            'Perform a partial 20-30% water exchange with oxygenated water',
            'Withhold feeding until surface piping resolves completely',
          ],
          expert_escalation: mortality24h > 10,
        })
      } else if (hasLesions) {
        setResult({
          condition: 'Motile Aeromonas Septicemia (MAS) / Columnaris Disease',
          confidence: 0.74,
          risk_level: 'high',
          supporting_evidence: [
            'Dermal ulceration & skin lesions present on fish body',
            'Lethargy and reduced feed conversion',
          ],
          missing_data: ['Microbiological skin scrape swab', 'pH and Nitrite levels'],
          recommended_steps: [
            'Isolate affected pond stock to prevent horizontal transmission',
            'Salt treatment dip (3-5 ppt) under veterinarian consultation',
            'Submit tissue sample for laboratory bacterial culture',
          ],
          expert_escalation: true,
        })
      } else {
        setResult({
          condition: 'Environmental Stress / Sub-optimal Water Quality',
          confidence: 0.62,
          risk_level: 'moderate',
          supporting_evidence: [
            'Nonspecific lethargy and reduced appetite without external lesions',
          ],
          missing_data: ['Dissolved Oxygen', 'pH', 'Ammonia', 'Turbidity'],
          recommended_steps: [
            'Conduct a full water quality testing panel',
            'Check pond bottom sludge accumulation',
            'Monitor fish feeding response over the next 12 hours',
          ],
          expert_escalation: false,
        })
      }
      setEvaluating(false)
    }, 600)
  }

  return (
    <div className="disease-page">
      <header className="page-header">
        <div>
          <span className="page-eyebrow">Diagnostic Safety Protocol</span>
          <h2>Disease & Health Assessment Tester</h2>
        </div>
        <p>
          Simulate clinical fish health symptoms to generate AquaDoc differential diagnosis reports and safety warnings.
        </p>
      </header>

      <div className="disease-container">
        <form className="disease-form" onSubmit={handleAssess}>
          <div className="form-section">
            <h3>1. Stock & Life Stage</h3>
            <div className="form-grid-2">
              <div className="field-group">
                <label htmlFor="species-select">Species</label>
                <select id="species-select" value={species} onChange={(e) => setSpecies(e.target.value)}>
                  <option value="Clarias gariepinus">African Catfish (Clarias gariepinus)</option>
                  <option value="Oreochromis niloticus">Nile Tilapia (Oreochromis niloticus)</option>
                  <option value="Heterobranchus longifilis">Vundu Catfish (Heterobranchus)</option>
                </select>
              </div>

              <div className="field-group">
                <label htmlFor="stage-select">Growth Stage</label>
                <select id="stage-select" value={stage} onChange={(e) => setStage(e.target.value)}>
                  <option value="Hatchling / Fry">Hatchling / Fry (&lt; 2g)</option>
                  <option value="Fingerling">Fingerling (2 - 10g)</option>
                  <option value="Juvenile">Juvenile (10 - 100g)</option>
                  <option value="Grow-out">Grow-out (&gt; 100g)</option>
                  <option value="Broodstock">Broodstock</option>
                </select>
              </div>
            </div>
          </div>

          <div className="form-section">
            <h3>2. Observed Symptoms</h3>
            <div className="symptom-checklist">
              {COMMON_SYMPTOMS.map((symptom) => {
                const checked = selectedSymptoms.includes(symptom)
                return (
                  <button
                    type="button"
                    key={symptom}
                    className={`symptom-chip ${checked ? 'symptom-chip--active' : ''}`}
                    onClick={() => toggleSymptom(symptom)}
                  >
                    <span className="checkbox-indicator">{checked ? '✓' : ''}</span>
                    <span>{symptom}</span>
                  </button>
                )
              })}
            </div>
          </div>

          <div className="form-section">
            <h3>3. Timeline & Mortality</h3>
            <div className="form-grid-2">
              <div className="field-group">
                <label htmlFor="duration-input">Duration (Days)</label>
                <input
                  id="duration-input"
                  type="number"
                  min="1"
                  max="30"
                  value={durationDays}
                  onChange={(e) => setDurationDays(Number(e.target.value))}
                />
              </div>

              <div className="field-group">
                <label htmlFor="mortality-input">Mortality in last 24h (fish count)</label>
                <input
                  id="mortality-input"
                  type="number"
                  min="0"
                  value={mortality24h}
                  onChange={(e) => setMortality24h(Number(e.target.value))}
                />
              </div>
            </div>
          </div>

          <div className="form-section">
            <h3>4. Key Environmental Data</h3>
            <div className="form-grid-2">
              <div className="field-group">
                <label htmlFor="temp-input">Water Temp (&deg;C)</label>
                <input
                  id="temp-input"
                  type="text"
                  placeholder="e.g. 29.5"
                  value={waterTemp}
                  onChange={(e) => setWaterTemp(e.target.value)}
                />
              </div>

              <div className="field-group">
                <label htmlFor="do-input">Dissolved Oxygen (mg/L)</label>
                <input
                  id="do-input"
                  type="text"
                  placeholder="Leave blank if unmeasured"
                  value={dissolvedOxygen}
                  onChange={(e) => setDissolvedOxygen(e.target.value)}
                />
              </div>
            </div>
          </div>

          <button type="submit" className="button button--primary button--lg" disabled={evaluating}>
            {evaluating ? 'Analyzing Symptoms…' : 'Run AquaDoc Health Assessment'}
          </button>
        </form>

        <div className="disease-report">
          {result ? (
            <div className="report-card">
              <div className="report-header">
                <div className="disclaimer-badge">AquaDoc Differential Assessment</div>
                <span className={`risk-tag risk-tag--${result.risk_level}`}>
                  {result.risk_level.toUpperCase()} RISK
                </span>
              </div>

              <h2 className="report-title">{result.condition}</h2>

              <div className="confidence-bar-container">
                <div className="confidence-label">
                  <span>AquaDoc Diagnostic Confidence</span>
                  <strong>{(result.confidence * 100).toFixed(0)}%</strong>
                </div>
                <div className="bar-track">
                  <div
                    className="bar-fill"
                    style={{ width: `${result.confidence * 100}%` }}
                  />
                </div>
              </div>

              {result.expert_escalation && (
                <div className="escalation-alert">
                  <svg viewBox="0 0 24 24" className="alert-icon" aria-hidden="true">
                    <path
                      d="M12 2L1 21H23L12 2ZM13 18H11V16H13V18ZM13 14H11V10H13V14Z"
                      fill="currentColor"
                    />
                  </svg>
                  <div>
                    <strong>Veterinarian Escalation Recommended</strong>
                    <p>High 24-hour mortality or bacterial symptom profile detected. Contact an aquatic health specialist immediately.</p>
                  </div>
                </div>
              )}

              <div className="report-block">
                <h4>Supporting Clinical Evidence</h4>
                <ul>
                  {result.supporting_evidence.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </div>

              {result.missing_data.length > 0 && (
                <div className="report-block report-block--warning">
                  <h4>Missing Parameters Needed for Conclusive Result</h4>
                  <ul>
                    {result.missing_data.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                </div>
              )}

              <div className="report-block">
                <h4>Recommended Immediate Actions</h4>
                <ol>
                  {result.recommended_steps.map((step) => (
                    <li key={step}>{step}</li>
                  ))}
                </ol>
              </div>

              <div className="report-footer">
                <small>
                  Notice: AquaDoc provides decision support recommendations grounded in approved aquaculture manuals. This assessment does not replace accredited laboratory disease testing.
                </small>
              </div>
            </div>
          ) : (
            <div className="report-placeholder">
              <div className="placeholder-content">
                <span className="ai-pulse-orb" aria-hidden="true" />
                <h3>Ready for Clinical Assessment</h3>
                <p>
                  Select the fish species, growth stage, observed symptoms, and environmental readings on the left, then click <strong>Run AquaDoc Health Assessment</strong>.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
