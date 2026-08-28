/**
 * Farm Simulator — Digital Twin of the Farm & Pond.
 *
 * Simulates multi-factor deterministic biological growth, dynamic FCR,
 * carrying capacity, environmental stress, and harvest economics across
 * continuous time horizons.
 */

import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppState } from '@/app/providers'
import {
  type DigitalTwinPondParams,
  type PondSystemType,
  type FishSpecies,
  simulateDigitalTwin,
  PRESET_SCENARIOS,
  SPECIES_PROFILES,
} from '@/utils/digitalTwinEngine'

export function FarmSimulatorPage() {
  const { farmForm, setFarmForm, setChatMode } = useAppState()
  const navigate = useNavigate()

  // Digital Twin parameters with defaults from farmForm or realistic commercial baseline
  const [params, setParams] = useState<DigitalTwinPondParams>({
    pondName: farmForm.pond_name || 'Tank 1 (Concrete Flow-Through)',
    systemType: 'concrete',
    waterVolumeM3: 20, // 20,000 Liters
    species: 'catfish',
    initialPopulation: Number(farmForm.population) || 1000,
    initialAvgWeightG: Number(farmForm.average_weight_g) || 150,
    targetHarvestWeightG: 1000, // 1kg table size
    waterTempC: Number(farmForm.temperature_c) || 28.5,
    dissolvedOxygenMgL: Number(farmForm.dissolved_oxygen_mg_l) || 5.5,
    ph: Number(farmForm.ph) || 7.2,
    ammoniaMgL: Number(farmForm.ammonia_mg_l) || 0.2,
    waterExchangePctPerDay: 40,
    aerationHoursPerDay: 10,
    dailyFeedRationG: Number(farmForm.daily_ration_g) || 4500,
    feedProteinPct: 42,
    feedCostPerKgNgn: 1750, // standard Nigerian 15kg feed rate (~26,250 NGN/bag)
    fishSellingPricePerKgNgn: 3200, // market farmgate price per kg
  })

  const [horizonDays, setHorizonDays] = useState<number>(60)
  const [activeScenario, setActiveScenario] = useState<string>('optimal')
  const [hoveredDay, setHoveredDay] = useState<number | null>(null)

  // Run the digital twin projection model
  const outcome = useMemo(() => {
    return simulateDigitalTwin(params, horizonDays)
  }, [params, horizonDays])

  // Apply scenario preset
  const handleScenarioChange = (scenarioKey: string) => {
    setActiveScenario(scenarioKey)
    const preset = PRESET_SCENARIOS[scenarioKey]
    if (preset) {
      setParams((prev) => ({ ...prev, ...preset.patch }))
    }
  }

  // Update a single parameter
  const updateParam = <K extends keyof DigitalTwinPondParams>(
    key: K,
    value: DigitalTwinPondParams[K]
  ) => {
    setParams((prev) => ({ ...prev, [key]: value }))
  }

  // Synchronize Digital Twin state to global farmForm for AquaDoc chat
  const handleSyncToAquaDoc = () => {
    setFarmForm({
      ...farmForm,
      pond_name: params.pondName,
      species: SPECIES_PROFILES[params.species].name,
      population: String(params.initialPopulation),
      average_weight_g: String(params.initialAvgWeightG),
      temperature_c: String(params.waterTempC),
      ph: String(params.ph),
      dissolved_oxygen_mg_l: String(params.dissolvedOxygenMgL),
      ammonia_mg_l: String(params.ammoniaMgL),
      daily_ration_g: String(params.dailyFeedRationG),
    })
    setChatMode('simulated_pond')
    navigate('/')
  }

  // SVG Chart geometry
  const maxWeight = Math.max(...outcome.steps.map((s) => s.avgWeightG), params.targetHarvestWeightG)
  const maxBiomass = Math.max(...outcome.steps.map((s) => s.biomassKg))
  const chartHeight = 220
  const chartWidth = 640

  const weightPoints = outcome.steps
    .map((s, i) => {
      const x = (i / (outcome.steps.length - 1)) * (chartWidth - 60) + 40
      const y = chartHeight - 30 - (s.avgWeightG / maxWeight) * (chartHeight - 60)
      return `${x},${y}`
    })
    .join(' ')

  const biomassPoints = outcome.steps
    .map((s, i) => {
      const x = (i / (outcome.steps.length - 1)) * (chartWidth - 60) + 40
      const y = chartHeight - 30 - (s.biomassKg / maxBiomass) * (chartHeight - 60)
      return `${x},${y}`
    })
    .join(' ')

  const activeStep = hoveredDay
    ? outcome.steps[hoveredDay - 1]
    : outcome.steps[outcome.steps.length - 1]

  return (
    <div className="digital-twin-page">
      {/* Page Header */}
      <header className="digital-twin__header">
        <div className="digital-twin__header-main">
          <div className="digital-twin__badge">
            <span className="live-dot" />
            <span>Digital Twin Engine v2.0</span>
          </div>
          <h1>Aquaculture Farm & Pond Digital Twin</h1>
          <p>
            Simulate and forecast growth trajectories, carrying capacity, dynamic FCR, and harvest
            economics based on consistent biological & environmental factors.
          </p>
        </div>

        <div className="digital-twin__header-actions">
          <button
            type="button"
            className="button button--primary"
            onClick={handleSyncToAquaDoc}
          >
            💬 Sync Twin to AquaDoc Chat
          </button>
        </div>
      </header>

      {/* Scenario Stress-Testing Selector */}
      <section className="twin-scenarios-bar">
        <span className="twin-scenarios-label">⚡ What-If Stress Scenarios:</span>
        <div className="twin-scenarios-list">
          {Object.entries(PRESET_SCENARIOS).map(([key, sc]) => (
            <button
              key={key}
              type="button"
              className={`scenario-chip ${activeScenario === key ? 'scenario-chip--active' : ''}`}
              onClick={() => handleScenarioChange(key)}
              title={sc.description}
            >
              <span className="scenario-chip__icon">{sc.icon}</span>
              <span className="scenario-chip__name">{sc.name}</span>
            </button>
          ))}
        </div>
      </section>

      {/* Main Grid: Parameters on Left, Twin Projections on Right */}
      <div className="twin-layout-grid">
        {/* Left Column: Interactive Parameters Form */}
        <aside className="twin-controls-panel">
          <div className="twin-panel-card">
            <h3>🏗️ Pond & Stock Profile</h3>

            <div className="twin-form-row">
              <label>System Type</label>
              <select
                value={params.systemType}
                onChange={(e) => updateParam('systemType', e.target.value as PondSystemType)}
                className="twin-select"
              >
                <option value="concrete">Concrete Flow-Through Tank</option>
                <option value="tarpaulin">Tarpaulin / Collapsible Tank</option>
                <option value="earthen">Earthen Pond</option>
                <option value="ras">Recirculating Aquaculture (RAS)</option>
              </select>
            </div>

            <div className="twin-form-grid-2">
              <div>
                <label>Water Volume (m³)</label>
                <input
                  type="number"
                  min="1"
                  max="10000"
                  value={params.waterVolumeM3}
                  onChange={(e) => updateParam('waterVolumeM3', Number(e.target.value) || 1)}
                  className="twin-input"
                />
                <span className="twin-hint">{(params.waterVolumeM3 * 1000).toLocaleString()} Liters</span>
              </div>
              <div>
                <label>Species</label>
                <select
                  value={params.species}
                  onChange={(e) => updateParam('species', e.target.value as FishSpecies)}
                  className="twin-select"
                >
                  <option value="catfish">African Catfish</option>
                  <option value="heteroclarias">Heteroclarias Hybrid</option>
                  <option value="tilapia">Nile Tilapia</option>
                </select>
              </div>
            </div>

            <div className="twin-form-grid-2">
              <div>
                <label>Initial Stock (count)</label>
                <input
                  type="number"
                  min="50"
                  max="500000"
                  value={params.initialPopulation}
                  onChange={(e) => updateParam('initialPopulation', Number(e.target.value) || 100)}
                  className="twin-input"
                />
              </div>
              <div>
                <label>Current Weight (g)</label>
                <input
                  type="number"
                  min="1"
                  max="5000"
                  value={params.initialAvgWeightG}
                  onChange={(e) => updateParam('initialAvgWeightG', Number(e.target.value) || 10)}
                  className="twin-input"
                />
              </div>
            </div>

            <div className="twin-form-row">
              <label>Target Harvest Weight (g)</label>
              <input
                type="number"
                min="100"
                max="5000"
                value={params.targetHarvestWeightG}
                onChange={(e) => updateParam('targetHarvestWeightG', Number(e.target.value) || 1000)}
                className="twin-input"
              />
            </div>
          </div>

          <div className="twin-panel-card">
            <h3>💧 Water Quality & Husbandry</h3>

            <div className="twin-form-grid-2">
              <div>
                <label>Water Temp (°C)</label>
                <input
                  type="number"
                  step="0.1"
                  min="15"
                  max="38"
                  value={params.waterTempC}
                  onChange={(e) => updateParam('waterTempC', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
              <div>
                <label>Dissolved Oxygen (mg/L)</label>
                <input
                  type="number"
                  step="0.1"
                  min="0.5"
                  max="14"
                  value={params.dissolvedOxygenMgL}
                  onChange={(e) => updateParam('dissolvedOxygenMgL', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
            </div>

            <div className="twin-form-grid-2">
              <div>
                <label>pH</label>
                <input
                  type="number"
                  step="0.1"
                  min="4"
                  max="11"
                  value={params.ph}
                  onChange={(e) => updateParam('ph', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
              <div>
                <label>Ammonia (mg/L)</label>
                <input
                  type="number"
                  step="0.05"
                  min="0"
                  max="5"
                  value={params.ammoniaMgL}
                  onChange={(e) => updateParam('ammoniaMgL', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
            </div>

            <div className="twin-form-grid-2">
              <div>
                <label>Water Exchange (%/day)</label>
                <input
                  type="number"
                  min="0"
                  max="200"
                  value={params.waterExchangePctPerDay}
                  onChange={(e) => updateParam('waterExchangePctPerDay', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
              <div>
                <label>Aeration (hours/day)</label>
                <input
                  type="number"
                  min="0"
                  max="24"
                  value={params.aerationHoursPerDay}
                  onChange={(e) => updateParam('aerationHoursPerDay', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
            </div>
          </div>

          <div className="twin-panel-card">
            <h3>💰 Economics & Pricing</h3>
            <div className="twin-form-grid-2">
              <div>
                <label>Feed Cost (₦ / kg)</label>
                <input
                  type="number"
                  min="500"
                  max="10000"
                  value={params.feedCostPerKgNgn}
                  onChange={(e) => updateParam('feedCostPerKgNgn', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
              <div>
                <label>Selling Price (₦ / kg)</label>
                <input
                  type="number"
                  min="1000"
                  max="15000"
                  value={params.fishSellingPricePerKgNgn}
                  onChange={(e) => updateParam('fishSellingPricePerKgNgn', Number(e.target.value))}
                  className="twin-input"
                />
              </div>
            </div>
          </div>
        </aside>

        {/* Right Column: Digital Twin Projections & Forecast Charts */}
        <main className="twin-results-panel">
          {/* Time Horizon Slider Bar */}
          <div className="twin-horizon-bar">
            <div className="twin-horizon-label">
              <span>📅 Simulation Horizon:</span>
              <strong>{horizonDays} Days</strong>
            </div>
            <div className="twin-horizon-presets">
              {[14, 30, 60, 90, 120].map((d) => (
                <button
                  key={d}
                  type="button"
                  className={`button button--tiny ${horizonDays === d ? 'button--primary' : 'button--ghost'}`}
                  onClick={() => setHorizonDays(d)}
                >
                  {d}d
                </button>
              ))}
            </div>
          </div>

          {/* KPI Summary Cards */}
          <div className="twin-kpi-grid">
            <div className="twin-kpi-card">
              <span className="twin-kpi-label">Projected Final Weight</span>
              <div className="twin-kpi-value">
                {outcome.finalAvgWeightG} <small>g</small>
              </div>
              <span className="twin-kpi-sub">
                Initial: {params.initialAvgWeightG}g (+{(outcome.finalAvgWeightG - params.initialAvgWeightG).toFixed(0)}g)
              </span>
            </div>

            <div className="twin-kpi-card">
              <span className="twin-kpi-label">Projected Biomass</span>
              <div className="twin-kpi-value">
                {outcome.finalBiomassKg.toLocaleString()} <small>kg</small>
              </div>
              <span className="twin-kpi-sub">
                Density: {(outcome.finalBiomassKg / params.waterVolumeM3).toFixed(1)} kg/m³
              </span>
            </div>

            <div className="twin-kpi-card">
              <span className="twin-kpi-label">Dynamic FCR & Feed</span>
              <div className="twin-kpi-value" style={{ color: outcome.overallFcr > 1.3 ? '#f59e0b' : '#10b981' }}>
                {outcome.overallFcr} <small>FCR</small>
              </div>
              <span className="twin-kpi-sub">
                {outcome.totalFeedBags15kg} bags (15kg)
              </span>
            </div>

            <div className="twin-kpi-card">
              <span className="twin-kpi-label">Survival & Days to 1kg</span>
              <div className="twin-kpi-value" style={{ color: outcome.survivalRatePct < 88 ? '#ef4444' : '#10b981' }}>
                {outcome.survivalRatePct}%
              </div>
              <span className="twin-kpi-sub">
                {outcome.daysToTargetHarvest ? `Target in ${outcome.daysToTargetHarvest} days` : `> ${horizonDays} days`}
              </span>
            </div>
          </div>

          {/* Interactive Trajectory Chart */}
          <div className="twin-chart-card">
            <div className="twin-chart-header">
              <div>
                <h3>📈 Digital Twin Growth & Biomass Forecast</h3>
                <p>Hover over the curve to inspect any day in the cycle.</p>
              </div>
              <div className="twin-chart-legend">
                <span className="legend-item"><span className="legend-dot legend-dot--cyan" /> Individual Weight (g)</span>
                <span className="legend-item"><span className="legend-dot legend-dot--purple" /> Total Biomass (kg)</span>
              </div>
            </div>

            {/* SVG Interactive Curve */}
            <div className="twin-svg-container">
              <svg
                viewBox={`0 0 ${chartWidth} ${chartHeight}`}
                className="twin-svg-chart"
                onMouseLeave={() => setHoveredDay(null)}
              >
                {/* Background Grid Lines */}
                <line x1="40" y1={chartHeight - 30} x2={chartWidth - 20} y2={chartHeight - 30} stroke="rgba(255,255,255,0.1)" />
                <line x1="40" y1={chartHeight / 2} x2={chartWidth - 20} y2={chartHeight / 2} stroke="rgba(255,255,255,0.05)" />
                <line x1="40" y1="30" x2={chartWidth - 20} y2="30" stroke="rgba(255,255,255,0.05)" />

                {/* Target Weight Threshold Line */}
                <line
                  x1="40"
                  y1={chartHeight - 30 - (params.targetHarvestWeightG / maxWeight) * (chartHeight - 60)}
                  x2={chartWidth - 20}
                  y2={chartHeight - 30 - (params.targetHarvestWeightG / maxWeight) * (chartHeight - 60)}
                  stroke="#f59e0b"
                  strokeDasharray="4 4"
                  strokeWidth="1.5"
                />
                <text
                  x={chartWidth - 25}
                  y={chartHeight - 35 - (params.targetHarvestWeightG / maxWeight) * (chartHeight - 60)}
                  fill="#f59e0b"
                  fontSize="10"
                  textAnchor="end"
                >
                  Target ({params.targetHarvestWeightG}g)
                </text>

                {/* Biomass Polyline */}
                <polyline
                  fill="none"
                  stroke="#a855f7"
                  strokeWidth="2.5"
                  strokeDasharray="2 2"
                  points={biomassPoints}
                />

                {/* Weight Polyline */}
                <polyline
                  fill="none"
                  stroke="#06b6d4"
                  strokeWidth="3.5"
                  points={weightPoints}
                />

                {/* Interactive Hover Nodes */}
                {outcome.steps.map((s, i) => {
                  const x = (i / (outcome.steps.length - 1)) * (chartWidth - 60) + 40
                  const y = chartHeight - 30 - (s.avgWeightG / maxWeight) * (chartHeight - 60)
                  return (
                    <circle
                      key={s.day}
                      cx={x}
                      cy={y}
                      r={hoveredDay === s.day ? 6 : 3}
                      fill={hoveredDay === s.day ? '#ffffff' : '#06b6d4'}
                      stroke="#06b6d4"
                      strokeWidth="2"
                      style={{ cursor: 'pointer', transition: 'all 0.15s ease' }}
                      onMouseEnter={() => setHoveredDay(s.day)}
                    />
                  )
                })}
              </svg>

              {/* Day Inspector Strip */}
              {activeStep && (
                <div className="twin-inspector-strip">
                  <div className="strip-item">
                    <span className="strip-label">Day {activeStep.day}</span>
                    <strong className="strip-val">{activeStep.avgWeightG} g</strong>
                  </div>
                  <div className="strip-item">
                    <span className="strip-label">Biomass</span>
                    <strong className="strip-val">{activeStep.biomassKg} kg</strong>
                  </div>
                  <div className="strip-item">
                    <span className="strip-label">Density</span>
                    <strong className="strip-val">{activeStep.stockingDensityKgM3} kg/m³</strong>
                  </div>
                  <div className="strip-item">
                    <span className="strip-label">Daily Feed</span>
                    <strong className="strip-val">{(activeStep.dailyFeedIntakeG / 1000).toFixed(2)} kg</strong>
                  </div>
                  <div className="strip-item">
                    <span className="strip-label">Daily SGR</span>
                    <strong className="strip-val">{activeStep.sgrPct}% / day</strong>
                  </div>
                  <div className="strip-item">
                    <span className="strip-label">Stress Index</span>
                    <strong
                      className="strip-val"
                      style={{ color: activeStep.stressIndexPct > 40 ? '#f59e0b' : '#10b981' }}
                    >
                      {activeStep.stressIndexPct}%
                    </strong>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Economics & Profitability Forecast Card */}
          <div className="twin-economics-card">
            <h3>💰 Projected Commercial Harvest Economics</h3>
            <div className="twin-econ-grid">
              <div className="econ-stat">
                <span className="econ-stat__label">Total Feed Investment</span>
                <strong className="econ-stat__val econ-stat__val--cost">
                  ₦ {outcome.totalFeedCostNgn.toLocaleString()}
                </strong>
                <small>{outcome.totalFeedBags15kg} bags (15kg @ ₦{params.feedCostPerKgNgn * 15}/bag)</small>
              </div>

              <div className="econ-stat">
                <span className="econ-stat__label">Est. Gross Revenue</span>
                <strong className="econ-stat__val econ-stat__val--rev">
                  ₦ {outcome.estimatedGrossRevenueNgn.toLocaleString()}
                </strong>
                <small>{outcome.finalBiomassKg.toFixed(0)} kg @ ₦{params.fishSellingPricePerKgNgn}/kg</small>
              </div>

              <div className="econ-stat">
                <span className="econ-stat__label">Projected Net Margin</span>
                <strong
                  className="econ-stat__val"
                  style={{ color: outcome.estimatedNetProfitNgn > 0 ? '#10b981' : '#ef4444' }}
                >
                  ₦ {outcome.estimatedNetProfitNgn.toLocaleString()} ({outcome.profitMarginPct}%)
                </strong>
                <small>Gross margin over feed costs</small>
              </div>
            </div>
          </div>

          {/* Dr. Fish Digital Twin Clinical Insights & Bottlenecks */}
          <div className="twin-insights-card">
            <div className="twin-insights-header">
              <span className="ai-orb">
                <span />
              </span>
              <div>
                <h3>🩺 Dr. Fish's Digital Twin Evaluation</h3>
                <p>Root-cause bottleneck analysis for your simulated conditions.</p>
              </div>
            </div>

            {outcome.bottlenecks.length > 0 && (
              <div className="twin-bottlenecks-list">
                <h4>⚠️ Detected Growth Bottlenecks:</h4>
                <ul>
                  {outcome.bottlenecks.map((b, idx) => (
                    <li key={idx}>{b}</li>
                  ))}
                </ul>
              </div>
            )}

            <div className="twin-recommendations-list">
              <h4>📋 Prescribed Husbandry Optimization:</h4>
              <ul>
                {outcome.recommendations.map((rec, idx) => (
                  <li key={idx}>{rec}</li>
                ))}
              </ul>
            </div>

            <div className="twin-actions-bar">
              <button
                type="button"
                className="button button--primary"
                onClick={handleSyncToAquaDoc}
              >
                Apply to Live AquaDoc Consultation →
              </button>
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
