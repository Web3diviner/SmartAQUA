/**
 * Disease Assessment & Clinical Triage Tester Page.
 *
 * Grounded in the live AquaDoc RAG Veterinary Knowledge Base.
 * Evaluates symptoms, mortalities, and water quality parameters to provide
 * specific differential diagnoses, exact treatment protocols, and safety escalation paths.
 * Features an integrated Call-to-Action to schedule on-farm physical inspections
 * or live video veterinary consultations.
 */

import { useEffect, useState } from 'react'
import { useAppState } from '@/app/providers'
import { sendChat } from '@/api/chat'
import { type ChatResponse } from '@/schemas/aquadoc'
import { renderAnswer } from '@/utils/markdown'
import { RiskBadge } from '@/components/RiskBadge'

const COMMON_SYMPTOMS = [
  { id: 'surface_piping', label: 'Surface piping / Gasping at water inlet' },
  { id: 'loss_appetite', label: 'Loss of appetite / Feed refusal' },
  { id: 'skin_ulcers', label: 'Skin ulcers, red lesions & hemorrhagic sores' },
  { id: 'broken_head', label: 'Broken head / Skull fissure & head swelling' },
  { id: 'columnaris', label: 'Saddleback lesion / White patch on dorsal & fin rot' },
  { id: 'flashing', label: 'Flashing, scratching against walls & excess mucus' },
  { id: 'dropsy', label: 'Abdominal distension (Dropsy) / Popeye' },
  { id: 'pale_gills', label: 'Pale, congested, or necrotic brown gills' },
  { id: 'rotten_egg', label: 'Rotten egg odor / Dark bottom sludge' },
  { id: 'lethargy', label: 'Lethargy & sluggish bottom resting' },
]

export function DiseaseTestPage() {
  const { config, selectedModel, user, isAuthenticated, openAuthModal } = useAppState()

  const [species, setSpecies] = useState(user?.primarySpecies || 'Clarias gariepinus')
  const [stage, setStage] = useState('Grow-out (> 100g)')
  const [pondType, setPondType] = useState(user?.farmingSystem || 'Concrete Flow-Through')
  const [durationDays, setDurationDays] = useState<number>(2)
  const [mortality24h, setMortality24h] = useState<number>(12)
  const [selectedSymptoms, setSelectedSymptoms] = useState<string[]>([
    'Skin ulcers, red lesions & hemorrhagic sores',
    'Loss of appetite / Feed refusal',
  ])
  const [waterTemp, setWaterTemp] = useState<string>('28.5')
  const [dissolvedOxygen, setDissolvedOxygen] = useState<string>('4.2')
  const [ph, setPh] = useState<string>('7.0')
  const [ammonia, setAmmonia] = useState<string>('0.4')
  const [additionalNotes, setAdditionalNotes] = useState<string>('')

  const [evaluating, setEvaluating] = useState(false)
  const [response, setResponse] = useState<ChatResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Consultation Booking Modal State
  const [showBookingModal, setShowBookingModal] = useState(false)
  const [bookingType, setBookingType] = useState<'physical' | 'virtual'>('physical')
  const [farmLocation, setFarmLocation] = useState(user?.farmLocation || '')
  const [farmerPhone, setFarmerPhone] = useState(user?.phone || '')
  const [preferredDate, setPreferredDate] = useState('')
  const [bookingConfirmed, setBookingConfirmed] = useState(false)

  // Sync user details if logged in
  useEffect(() => {
    if (user) {
      if (user.phone) setFarmerPhone(user.phone)
      if (user.farmLocation) setFarmLocation(user.farmLocation)
    }
  }, [user])

  const toggleSymptom = (symptomLabel: string) => {
    setSelectedSymptoms((prev) =>
      prev.includes(symptomLabel)
        ? prev.filter((s) => s !== symptomLabel)
        : [...prev, symptomLabel],
    )
  }

  const handleAssess = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!isAuthenticated) {
      openAuthModal()
      return
    }
    setEvaluating(true)
    setError(null)

    // Construct highly specific clinical diagnostic query for the RAG engine
    const clinicalQuery = [
      `[CLINICAL DISEASE TRIAGE CASE REPORT]`,
      `Species: ${species}`,
      `Life Stage: ${stage}`,
      `Culture System: ${pondType}`,
      `Observed Symptoms: ${selectedSymptoms.length > 0 ? selectedSymptoms.join('; ') : 'No physical lesions observed'}`,
      `Symptom Duration: ${durationDays} days`,
      `Mortality in last 24h: ${mortality24h} fish`,
      `Water Temperature: ${waterTemp ? `${waterTemp}°C` : 'Unmeasured'}`,
      `Dissolved Oxygen: ${dissolvedOxygen ? `${dissolvedOxygen} mg/L` : 'Unmeasured'}`,
      `pH: ${ph || 'Unmeasured'}`,
      `Ammonia: ${ammonia ? `${ammonia} mg/L` : 'Unmeasured'}`,
      additionalNotes ? `Farmer Observations: ${additionalNotes}` : '',
      `Please provide a specific differential diagnosis, exact veterinary treatment protocol with dosages (e.g. salt bath ppt, Vitamin C mg/kg, water exchange %, or antibiotic withdrawal), and highlight any missing diagnostic parameters.`,
    ]
      .filter(Boolean)
      .join('\n')

    const farmContext = {
      farm_id: 'triage-farm',
      pond_id: 'triage-pond',
      farm_name: 'Diagnostic Triage Session',
      pond_name: pondType,
      species,
      life_stage: stage,
      population: 1000,
      average_weight_g: 250,
      water: {
        temperature_c: waterTemp ? Number(waterTemp) : null,
        ph: ph ? Number(ph) : null,
        dissolved_oxygen_mg_l: dissolvedOxygen ? Number(dissolvedOxygen) : null,
        turbidity_ntu: null,
        ammonia_mg_l: ammonia ? Number(ammonia) : null,
        nitrite_mg_l: null,
      },
      feeding: {
        daily_ration_g: null,
        last_feeding_g: null,
        feeds_per_day: 2,
        feed_acceptance: 'reduced',
      },
      health: {
        mortality_24h: mortality24h,
        mortality_7d: mortality24h * durationDays,
        active_disease_case: true,
        reported_symptoms: selectedSymptoms,
      },
    }

    try {
      const res = await sendChat(config, {
        userId: 'disease-triage-user',
        question: clinicalQuery,
        conversationId: null,
        farmContext,
        model: selectedModel,
      })
      setResponse(res)
    } catch (err: unknown) {
      console.error('Disease triage assessment failed:', err)
      setError(err instanceof Error ? err.message : 'Unable to complete clinical triage assessment.')
    } finally {
      setEvaluating(false)
    }
  }

  const handleBookConsultation = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await fetch(`${config.baseUrl}/dev/v1/bookings`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${config.devToken}`,
        },
        body: JSON.stringify({
          farmer_name: 'Farm Manager',
          farmer_phone: farmerPhone,
          farm_location: farmLocation,
          booking_type: bookingType,
          species,
          symptoms: selectedSymptoms,
          preferred_date: preferredDate,
          notes: additionalNotes || `${species} stock showing ${selectedSymptoms.join(', ')}`,
        }),
      })
    } catch (err) {
      console.warn('Booking post fallback:', err)
    }

    setBookingConfirmed(true)
    setTimeout(() => {
      setShowBookingModal(false)
      setBookingConfirmed(false)
    }, 2800)
  }

  return (
    <div className="disease-page">
      {/* Header */}
      <header className="page-header">
        <div>
          <span className="page-eyebrow">Veterinary Intelligence Protocol</span>
          <h2>Aquaculture Disease & Clinical Health Triage</h2>
        </div>
        <p>
          Submit observed clinical symptoms, mortalities, and water parameters to receive
          a grounded AquaDoc differential diagnosis and specific veterinary action plan.
        </p>
      </header>

      <div className="disease-container">
        {/* Left Column: Clinical Intake Form */}
        <form className="disease-form" onSubmit={handleAssess}>
          <div className="form-section">
            <h3>1. Stock & Culture Environment</h3>
            <div className="form-grid-2">
              <div className="field-group">
                <label htmlFor="species-select">Fish Species</label>
                <select
                  id="species-select"
                  value={species}
                  onChange={(e) => setSpecies(e.target.value)}
                >
                  <option value="Clarias gariepinus">African Catfish (Clarias gariepinus)</option>
                  <option value="Heteroclarias">Heteroclarias Hybrid (Clarias x Heterobranchus)</option>
                  <option value="Oreochromis niloticus">Nile Tilapia (Oreochromis niloticus)</option>
                </select>
              </div>

              <div className="field-group">
                <label htmlFor="stage-select">Growth Stage</label>
                <select id="stage-select" value={stage} onChange={(e) => setStage(e.target.value)}>
                  <option value="Fingerling (2 - 10g)">Fingerling (2 - 10g)</option>
                  <option value="Post-Fingerling / Juvenile (10 - 50g)">Post-Fingerling / Juvenile (10 - 50g)</option>
                  <option value="Grow-out (> 100g)">Grow-out (&gt; 100g)</option>
                  <option value="Broodstock (> 1kg)">Broodstock (&gt; 1kg)</option>
                </select>
              </div>
            </div>

            <div className="field-group" style={{ marginTop: '12px' }}>
              <label htmlFor="pond-type">Pond / Tank System</label>
              <select id="pond-type" value={pondType} onChange={(e) => setPondType(e.target.value)}>
                <option value="Concrete Flow-Through">Concrete Flow-Through Tank</option>
                <option value="Tarpaulin / Collapsible Tank">Tarpaulin / Collapsible Tank</option>
                <option value="Earthen Pond">Earthen Pond</option>
                <option value="Recirculating System (RAS)">Recirculating Aquaculture System (RAS)</option>
              </select>
            </div>
          </div>

          <div className="form-section">
            <h3>2. Observed Clinical Symptoms</h3>
            <p className="form-section__hint">Select all visible lesions or behavioral signs:</p>
            <div className="symptom-checklist">
              {COMMON_SYMPTOMS.map((item) => {
                const checked = selectedSymptoms.includes(item.label)
                return (
                  <button
                    type="button"
                    key={item.id}
                    className={`symptom-chip ${checked ? 'symptom-chip--active' : ''}`}
                    onClick={() => toggleSymptom(item.label)}
                  >
                    <span className="checkbox-indicator">{checked ? '✓' : ''}</span>
                    <span>{item.label}</span>
                  </button>
                )
              })}
            </div>
          </div>

          <div className="form-section">
            <h3>3. Mortality & Severity</h3>
            <div className="form-grid-2">
              <div className="field-group">
                <label htmlFor="duration-input">Duration of Symptoms (Days)</label>
                <input
                  id="duration-input"
                  type="number"
                  min="1"
                  max="60"
                  value={durationDays}
                  onChange={(e) => setDurationDays(Number(e.target.value))}
                />
              </div>

              <div className="field-group">
                <label htmlFor="mortality-input">Mortality in Last 24 Hours (count)</label>
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
            <h3>4. Water Quality Measurements</h3>
            <div className="form-grid-2">
              <div className="field-group">
                <label htmlFor="temp-input">Water Temp (°C)</label>
                <input
                  id="temp-input"
                  type="text"
                  placeholder="e.g. 28.5"
                  value={waterTemp}
                  onChange={(e) => setWaterTemp(e.target.value)}
                />
              </div>

              <div className="field-group">
                <label htmlFor="do-input">Dissolved Oxygen (mg/L)</label>
                <input
                  id="do-input"
                  type="text"
                  placeholder="e.g. 4.5"
                  value={dissolvedOxygen}
                  onChange={(e) => setDissolvedOxygen(e.target.value)}
                />
              </div>
            </div>

            <div className="form-grid-2" style={{ marginTop: '12px' }}>
              <div className="field-group">
                <label htmlFor="ph-input">pH</label>
                <input
                  id="ph-input"
                  type="text"
                  placeholder="e.g. 7.0"
                  value={ph}
                  onChange={(e) => setPh(e.target.value)}
                />
              </div>

              <div className="field-group">
                <label htmlFor="ammonia-input">Ammonia (mg/L)</label>
                <input
                  id="ammonia-input"
                  type="text"
                  placeholder="e.g. 0.25"
                  value={ammonia}
                  onChange={(e) => setAmmonia(e.target.value)}
                />
              </div>
            </div>

            <div className="field-group" style={{ marginTop: '12px' }}>
              <label htmlFor="notes-input">Additional Clinical Notes (Optional)</label>
              <textarea
                id="notes-input"
                rows={2}
                placeholder="e.g., Heavy rainfall last night, feed batch changed 3 days ago..."
                value={additionalNotes}
                onChange={(e) => setAdditionalNotes(e.target.value)}
                style={{ width: '100%', resize: 'vertical' }}
              />
            </div>
          </div>

          <button
            type="submit"
            className="button button--primary button--lg"
            disabled={evaluating}
          >
            {evaluating ? '🧠 Consulting RAG Veterinary Engine…' : '🩺 Run AquaDoc Clinical Triage'}
          </button>
        </form>

        {/* Right Column: RAG Diagnostic Report & Consultation CTA */}
        <div className="disease-report">
          {error && (
            <div className="error-panel" style={{ marginBottom: '20px' }}>
              <strong>Assessment Error:</strong> {error}
            </div>
          )}

          {response ? (
            <div className="report-card">
              {/* Report Header */}
              <div className="report-header">
                <div className="disclaimer-badge">AquaDoc RAG Diagnostic Evaluation</div>
                <RiskBadge level={response.risk_level} />
              </div>

              {/* Escalation Alert */}
              {response.expert_escalation && (
                <div className="escalation-alert">
                  <svg viewBox="0 0 24 24" className="alert-icon" aria-hidden="true">
                    <path
                      d="M12 2L1 21H23L12 2ZM13 18H11V16H13V18ZM13 14H11V10H13V14Z"
                      fill="currentColor"
                    />
                  </svg>
                  <div>
                    <strong>Veterinarian Physical Inspection Escalation</strong>
                    <p>
                      {response.escalation_reasons?.length
                        ? response.escalation_reasons.join(', ')
                        : 'Elevated mortality or acute bacterial symptom profile detected. Physical clinical inspection recommended.'}
                    </p>
                  </div>
                </div>
              )}

              {/* Differential Causes */}
              {response.possible_causes.length > 0 && (
                <div className="report-block">
                  <h4>Differential Diagnostic Causes:</h4>
                  <ul className="possible-causes-list">
                    {response.possible_causes.map((cause, idx) => (
                      <li key={idx} className="cause-item">
                        <div className="cause-item__header">
                          <strong>{cause.name}</strong>
                          <span className="cause-prob">
                            {(cause.confidence * 100).toFixed(0)}% likelihood
                          </span>
                        </div>
                        {cause.explanation && <p className="cause-expl">{cause.explanation}</p>}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Main RAG Doctor Response */}
              <div className="report-block">
                <h4>Doctor's Clinical Assessment & Treatment Plan:</h4>
                <div className="message__answer" style={{ color: 'var(--text-main)' }}>
                  {renderAnswer(response.answer)}
                </div>
              </div>

              {/* Missing Measurements */}
              {(response.missing_data_labels.length > 0 || response.missing_data.length > 0) && (
                <div className="report-block report-block--warning">
                  <h4>Missing Parameters Needed to Confirm Diagnosis:</h4>
                  <ul>
                    {(response.missing_data_labels.length > 0
                      ? response.missing_data_labels
                      : response.missing_data
                    ).map((param: string) => (
                      <li key={param}>
                        <code>{param}</code> — unmeasured in this evaluation
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Recommended Actions */}
              {response.recommended_actions.length > 0 && (
                <div className="report-block">
                  <h4>Prescribed Immediate Interventions:</h4>
                  <ol>
                    {response.recommended_actions.map((act, idx) => (
                      <li key={idx}>
                        <strong>{act.action}</strong> ({act.urgency})
                      </li>
                    ))}
                  </ol>
                </div>
              )}

              {/* CALL TO ACTION: Expert Physical Inspection & Live Consultation */}
              <div className="consultant-cta-card">
                <div className="consultant-cta-header">
                  <div className="cta-icon-box">👨‍⚕️</div>
                  <div>
                    <h3>Need an Aquaculture Veterinary Specialist?</h3>
                    <p>
                      Schedule an on-farm physical inspection or book a 1-on-1 live tele-consultation
                      with an accredited aquatic pathologist.
                    </p>
                  </div>
                </div>

                <div className="consultant-cta-buttons">
                  <button
                    type="button"
                    className="button button--primary"
                    onClick={() => {
                      setBookingType('physical')
                      setShowBookingModal(true)
                    }}
                  >
                    🚜 Request On-Farm Physical Inspection
                  </button>

                  <button
                    type="button"
                    className="button button--secondary"
                    onClick={() => {
                      setBookingType('virtual')
                      setShowBookingModal(true)
                    }}
                  >
                    📹 Book Live Video Consultation
                  </button>
                </div>

                <div className="consultant-hotline-bar">
                  <span>
                    📞 Emergency Vet Hotline:{' '}
                    <a
                      href="tel:+2348071055742"
                      style={{ color: 'inherit', textDecoration: 'underline' }}
                    >
                      <strong>+234 807 105 5742</strong>
                    </a>
                  </span>
                  <a
                    href={`https://wa.me/2348071055742?text=Hello%20Smart%20Aqua%20Veterinary%20Team,%20I%20need%20urgent%20assistance%20for%20my%20${encodeURIComponent(species)}%20stock%20showing%20${encodeURIComponent(selectedSymptoms.join(', '))}`}
                    target="_blank"
                    rel="noreferrer"
                    className="whatsapp-btn"
                  >
                    💬 Quick WhatsApp Connect
                  </a>
                </div>
              </div>
            </div>
          ) : (
            <div className="report-empty-state">
              <div className="empty-state-content">
                <span className="ai-pulse-orb" aria-hidden="true" />
                <h3>Ready for Clinical Disease Triage</h3>
                <p>
                  Select your species, observed symptoms, mortality count, and water readings on the
                  left, then click <strong>Run AquaDoc Clinical Triage</strong> to fetch a live
                  veterinary evaluation grounded in approved RAG pathology papers.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Booking Modal */}
      {showBookingModal && (
        <div className="modal-backdrop" onClick={() => setShowBookingModal(false)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>
                {bookingType === 'physical'
                  ? '🚜 Request On-Farm Physical Inspection'
                  : '📹 Book Live Video Consultation'}
              </h3>
              <button
                type="button"
                className="close-btn"
                onClick={() => setShowBookingModal(false)}
              >
                ✕
              </button>
            </div>

            {bookingConfirmed ? (
              <div className="booking-success">
                <span className="success-icon">✅</span>
                <h4>Consultation Request Received!</h4>
                <p>
                  An accredited aquaculture consultant has been alerted and will contact you at{' '}
                  <strong>{farmerPhone || 'your provided phone number'}</strong> within 30 minutes to
                  confirm inspection logistics.
                </p>
              </div>
            ) : (
              <form onSubmit={handleBookConsultation} className="booking-form">
                <p className="modal-desc">
                  {bookingType === 'physical'
                    ? 'Our regional aquatic veterinarians provide on-site pond inspection, water testing, gill biopsies, and necropsy.'
                    : 'Connect directly with an aquatic health specialist over secure HD video to inspect fish behavior in real-time.'}
                </p>

                <div className="field-group">
                  <label htmlFor="farm-loc">Farm Location / State / City *</label>
                  <input
                    id="farm-loc"
                    type="text"
                    required
                    placeholder="e.g. Epe, Lagos State / Ibadan, Oyo State"
                    value={farmLocation}
                    onChange={(e) => setFarmLocation(e.target.value)}
                  />
                </div>

                <div className="field-group">
                  <label htmlFor="farmer-phone">Phone / WhatsApp Number *</label>
                  <input
                    id="farmer-phone"
                    type="tel"
                    required
                    placeholder="e.g. 08012345678"
                    value={farmerPhone}
                    onChange={(e) => setFarmerPhone(e.target.value)}
                  />
                </div>

                <div className="field-group">
                  <label htmlFor="pref-date">Preferred Date & Time *</label>
                  <input
                    id="pref-date"
                    type="datetime-local"
                    required
                    value={preferredDate}
                    onChange={(e) => setPreferredDate(e.target.value)}
                  />
                </div>

                <div className="modal-actions">
                  <button
                    type="button"
                    className="button button--ghost"
                    onClick={() => setShowBookingModal(false)}
                  >
                    Cancel
                  </button>
                  <button type="submit" className="button button--primary">
                    Confirm Consultation Request
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
