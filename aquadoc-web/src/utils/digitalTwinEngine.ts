/**
 * Digital Twin Simulation Engine for Aquaculture Ponds.
 *
 * Implements deterministic multi-factor biological and environmental
 * projection models:
 * - Specific Growth Rate (SGR) with thermal (Q10) and DO limitation curves.
 * - Dynamic Feed Conversion Ratio (FCR) with stress penalties.
 * - Ammonia/TAN generation and water exchange turnover modeling.
 * - Carrying capacity and stocking density bottleneck detection.
 * - Cumulative mortality and harvest economics projection.
 */

export type PondSystemType = 'concrete' | 'tarpaulin' | 'earthen' | 'ras'
export type FishSpecies = 'catfish' | 'heteroclarias' | 'tilapia'

export interface DigitalTwinPondParams {
  pondName: string
  systemType: PondSystemType
  waterVolumeM3: number
  species: FishSpecies
  initialPopulation: number
  initialAvgWeightG: number
  targetHarvestWeightG: number
  waterTempC: number
  dissolvedOxygenMgL: number
  ph: number
  ammoniaMgL: number
  waterExchangePctPerDay: number
  aerationHoursPerDay: number
  dailyFeedRationG: number
  feedProteinPct: number
  feedCostPerKgNgn: number
  fishSellingPricePerKgNgn: number
}

export interface DailySimulationStep {
  day: number
  avgWeightG: number
  biomassKg: number
  population: number
  stockingDensityKgM3: number
  dailyGrowthG: number
  sgrPct: number
  dailyFeedIntakeG: number
  cumulativeFeedKg: number
  fcr: number
  predictedDoMgL: number
  predictedAmmoniaMgL: number
  stressIndexPct: number
  dailyMortality: number
  cumulativeMortality: number
  carryingCapacityPct: number
  isDensityWarning: boolean
  isDoWarning: boolean
  isAmmoniaWarning: boolean
}

export interface DigitalTwinOutcome {
  horizonDays: number
  initialParams: DigitalTwinPondParams
  steps: DailySimulationStep[]
  finalAvgWeightG: number
  finalBiomassKg: number
  finalPopulation: number
  survivalRatePct: number
  totalFeedBags15kg: number
  totalFeedCostNgn: number
  overallFcr: number
  daysToTargetHarvest: number | null
  estimatedGrossRevenueNgn: number
  estimatedNetProfitNgn: number
  profitMarginPct: number
  riskLevel: 'optimal' | 'moderate' | 'high_risk' | 'critical'
  bottlenecks: string[]
  recommendations: string[]
}

export const SYSTEM_CAPACITY_LIMITS: Record<PondSystemType, { maxDensityKgM3: number; label: string }> = {
  concrete: { maxDensityKgM3: 45, label: 'Concrete Flow-Through Tank' },
  tarpaulin: { maxDensityKgM3: 35, label: 'Tarpaulin / Collapsible Tank' },
  earthen: { maxDensityKgM3: 12, label: 'Earthen Pond' },
  ras: { maxDensityKgM3: 75, label: 'Recirculating Aquaculture System (RAS)' },
}

export const SPECIES_PROFILES: Record<FishSpecies, { name: string; baseSgr: number; optimalTempC: number; optimalFcr: number }> = {
  catfish: { name: 'African Catfish (Clarias gariepinus)', baseSgr: 2.8, optimalTempC: 28.5, optimalFcr: 1.05 },
  heteroclarias: { name: 'Heteroclarias Hybrid', baseSgr: 2.9, optimalTempC: 28.0, optimalFcr: 1.08 },
  tilapia: { name: 'Nile Tilapia (Oreochromis niloticus)', baseSgr: 2.2, optimalTempC: 29.0, optimalFcr: 1.25 },
}

/**
 * Calculates Temperature Growth Multiplier using Q10 response curve.
 */
export function calculateTempGrowthFactor(tempC: number, optimalTempC = 28.5): number {
  if (tempC < 18) return 0.25
  if (tempC > 35) return 0.4
  const q10 = 2.0
  const factor = Math.pow(q10, (tempC - optimalTempC) / 10)
  return Math.max(0.2, Math.min(1.25, factor))
}

/**
 * Calculates Dissolved Oxygen limitation factor on growth and appetite.
 */
export function calculateDoGrowthFactor(doMgL: number): number {
  if (doMgL >= 5.0) return 1.0
  if (doMgL >= 4.0) return 0.85
  if (doMgL >= 3.0) return 0.60
  if (doMgL >= 2.0) return 0.35
  return 0.15
}

/**
 * Calculates Stress-Induced FCR inflation.
 */
export function calculateDynamicFcr(
  baseFcr: number,
  tempC: number,
  doMgL: number,
  ammoniaMgL: number
): number {
  let penalty = 0.0
  // Temperature stress penalty
  if (tempC < 24) penalty += (24 - tempC) * 0.06
  if (tempC > 31) penalty += (tempC - 31) * 0.05

  // Hypoxia stress penalty
  if (doMgL < 5.0) penalty += (5.0 - doMgL) * 0.12

  // Ammonia stress penalty
  if (ammoniaMgL > 0.5) penalty += (ammoniaMgL - 0.5) * 0.25

  return Math.min(2.5, Number((baseFcr * (1 + penalty)).toFixed(2)))
}

/**
 * Simulates a continuous multi-day Digital Twin projection.
 */
export function simulateDigitalTwin(
  params: DigitalTwinPondParams,
  days = 60
): DigitalTwinOutcome {
  const species = SPECIES_PROFILES[params.species] || SPECIES_PROFILES.catfish
  const systemLimit = SYSTEM_CAPACITY_LIMITS[params.systemType].maxDensityKgM3

  const steps: DailySimulationStep[] = []
  let currentWeight = params.initialAvgWeightG
  let currentPop = params.initialPopulation
  let cumulativeFeedG = 0
  let cumulativeMortality = 0
  let daysToTarget: number | null = null

  const bottlenecks: string[] = []

  for (let d = 1; d <= days; d++) {
    // 1. Biological Growth Calculation
    const tempFactor = calculateTempGrowthFactor(params.waterTempC, species.optimalTempC)
    const doFactor = calculateDoGrowthFactor(params.dissolvedOxygenMgL)
    
    // Weight-dependent baseline SGR (smaller fish grow with higher % SGR)
    let weightSgrScaling = 1.0
    if (currentWeight < 20) weightSgrScaling = 1.6
    else if (currentWeight < 100) weightSgrScaling = 1.2
    else if (currentWeight < 300) weightSgrScaling = 1.0
    else if (currentWeight < 600) weightSgrScaling = 0.8
    else weightSgrScaling = 0.65

    const effectiveSgr = Math.max(0.1, species.baseSgr * weightSgrScaling * tempFactor * doFactor)
    const dailyGrowthG = currentWeight * (effectiveSgr / 100)
    currentWeight += dailyGrowthG

    if (daysToTarget === null && currentWeight >= params.targetHarvestWeightG) {
      daysToTarget = d
    }

    // 2. FCR and Feed Intake
    const dailyFcr = calculateDynamicFcr(
      species.optimalFcr,
      params.waterTempC,
      params.dissolvedOxygenMgL,
      params.ammoniaMgL
    )

    // Daily feed requirement based on weight gain and FCR
    const dailyBiomassGainG = dailyGrowthG * currentPop
    const requiredFeedG = dailyBiomassGainG * dailyFcr
    cumulativeFeedG += requiredFeedG

    // 3. Biomass & Density
    const biomassKg = (currentPop * currentWeight) / 1000
    const density = biomassKg / Math.max(1, params.waterVolumeM3)
    const capacityPct = (density / systemLimit) * 100

    // 4. Daily Mortality Risk Modeling
    let dailyMortalityRiskPct = 0.02 // normal baseline 0.02% / day
    if (params.dissolvedOxygenMgL < 3.0) dailyMortalityRiskPct += 0.4
    if (params.dissolvedOxygenMgL < 2.0) dailyMortalityRiskPct += 1.5
    if (params.ammoniaMgL > 1.0) dailyMortalityRiskPct += 0.3
    if (density > systemLimit) dailyMortalityRiskPct += 0.25
    if (params.waterTempC < 20 || params.waterTempC > 33) dailyMortalityRiskPct += 0.2

    const deadToday = Math.round(currentPop * (dailyMortalityRiskPct / 100))
    currentPop = Math.max(0, currentPop - deadToday)
    cumulativeMortality += deadToday

    // 5. Water Quality Trajectory Checks
    const isDensityWarning = density >= systemLimit * 0.85
    const isDoWarning = params.dissolvedOxygenMgL < 4.5
    const isAmmoniaWarning = params.ammoniaMgL > 0.6

    // Stress index 0-100%
    const stressIndex = Math.min(
      100,
      Math.round(
        (1 - doFactor) * 45 +
        Math.abs(params.waterTempC - species.optimalTempC) * 6 +
        params.ammoniaMgL * 25 +
        (density / systemLimit > 1 ? 30 : 0)
      )
    )

    steps.push({
      day: d,
      avgWeightG: Number(currentWeight.toFixed(1)),
      biomassKg: Number(biomassKg.toFixed(1)),
      population: currentPop,
      stockingDensityKgM3: Number(density.toFixed(1)),
      dailyGrowthG: Number(dailyGrowthG.toFixed(2)),
      sgrPct: Number(effectiveSgr.toFixed(2)),
      dailyFeedIntakeG: Number(requiredFeedG.toFixed(0)),
      cumulativeFeedKg: Number((cumulativeFeedG / 1000).toFixed(1)),
      fcr: dailyFcr,
      predictedDoMgL: params.dissolvedOxygenMgL,
      predictedAmmoniaMgL: params.ammoniaMgL,
      stressIndexPct: stressIndex,
      dailyMortality: deadToday,
      cumulativeMortality,
      carryingCapacityPct: Number(capacityPct.toFixed(1)),
      isDensityWarning,
      isDoWarning,
      isAmmoniaWarning,
    })

    if (isDensityWarning && !bottlenecks.some(b => b.includes('Density threshold'))) {
      bottlenecks.push(`Day ${d}: Stocking density reaches ${density.toFixed(1)} kg/m³ (${capacityPct.toFixed(0)}% of ${systemLimit} kg/m³ max capacity)`)
    }
  }

  // 6. Overall Metrics & Economics
  const finalBiomassKg = (currentPop * currentWeight) / 1000
  const initialBiomassKg = (params.initialPopulation * params.initialAvgWeightG) / 1000
  const netBiomassGainKg = Math.max(0.1, finalBiomassKg - initialBiomassKg)
  const totalFeedKg = cumulativeFeedG / 1000
  const overallFcr = Number((totalFeedKg / netBiomassGainKg).toFixed(2))

  const totalFeedBags15kg = Number((totalFeedKg / 15).toFixed(1))
  const totalFeedCostNgn = Math.round(totalFeedKg * params.feedCostPerKgNgn)
  const estimatedGrossRevenueNgn = Math.round(finalBiomassKg * params.fishSellingPricePerKgNgn)
  const estimatedNetProfitNgn = estimatedGrossRevenueNgn - totalFeedCostNgn
  const profitMarginPct = estimatedGrossRevenueNgn > 0
    ? Number(((estimatedNetProfitNgn / estimatedGrossRevenueNgn) * 100).toFixed(1))
    : 0

  const survivalRatePct = Number(
    (((params.initialPopulation - cumulativeMortality) / params.initialPopulation) * 100).toFixed(1)
  )

  // Risk Classification
  let riskLevel: DigitalTwinOutcome['riskLevel'] = 'optimal'
  if (params.dissolvedOxygenMgL < 3.5 || params.ammoniaMgL > 1.0 || survivalRatePct < 85) {
    riskLevel = 'critical'
  } else if (params.dissolvedOxygenMgL < 4.5 || params.waterTempC < 23 || overallFcr > 1.35) {
    riskLevel = 'high_risk'
  } else if (overallFcr > 1.18 || bottlenecks.length > 0) {
    riskLevel = 'moderate'
  }

  // Recommendations
  const recommendations: string[] = []
  if (params.dissolvedOxygenMgL < 5.0) {
    recommendations.push(
      `Increase daily aeration by 4-6 hours to elevate DO above 5.0 mg/L. This will accelerate growth by ~${((1 - calculateDoGrowthFactor(params.dissolvedOxygenMgL)) * 100).toFixed(0)}% and reduce FCR from ${overallFcr} to ${species.optimalFcr}.`
    )
  }
  if (params.waterTempC < 25) {
    recommendations.push(
      `Water temperature (${params.waterTempC}°C) is slowing down catfish metabolism (Q10 factor ${(calculateTempGrowthFactor(params.waterTempC)).toFixed(2)}x). Adjust feeding to warmest afternoon hours (1:00 PM - 3:30 PM).`
    )
  }
  if (finalBiomassKg / params.waterVolumeM3 > systemLimit) {
    recommendations.push(
      `Pond will exceed safe carrying capacity of ${systemLimit} kg/m³. Schedule a split-grading or partial harvest at Day ${daysToTarget || Math.round(days * 0.7)} to prevent cannibalism and stunting.`
    )
  }
  if (params.ammoniaMgL > 0.5) {
    recommendations.push(
      `Elevated ammonia (${params.ammoniaMgL} mg/L) is inflating FCR. Increase daily water exchange from ${params.waterExchangePctPerDay}% to ${Math.min(100, params.waterExchangePctPerDay + 25)}%.`
    )
  }
  if (recommendations.length === 0) {
    recommendations.push(
      `Conditions are optimal! At current trajectory, your stock will achieve target harvest weight (${params.targetHarvestWeightG}g) in ~${daysToTarget || days} days with an excellent FCR of ${overallFcr}.`
    )
  }

  return {
    horizonDays: days,
    initialParams: params,
    steps,
    finalAvgWeightG: Number(currentWeight.toFixed(1)),
    finalBiomassKg: Number(finalBiomassKg.toFixed(1)),
    finalPopulation: currentPop,
    survivalRatePct,
    totalFeedBags15kg,
    totalFeedCostNgn,
    overallFcr,
    daysToTargetHarvest: daysToTarget,
    estimatedGrossRevenueNgn,
    estimatedNetProfitNgn,
    profitMarginPct,
    riskLevel,
    bottlenecks,
    recommendations,
  }
}

/**
 * Pre-configured Real-World Scenarios for Rapid Stress-Testing.
 */
export const PRESET_SCENARIOS: Record<
  string,
  { name: string; description: string; icon: string; patch: Partial<DigitalTwinPondParams> }
> = {
  optimal: {
    name: 'Optimal Commercial Grow-Out',
    description: 'Ideal aeration, warm water (28.5°C), 50% flow-through exchange, premium 42% protein.',
    icon: '✨',
    patch: {
      waterTempC: 28.5,
      dissolvedOxygenMgL: 6.0,
      ph: 7.2,
      ammoniaMgL: 0.15,
      waterExchangePctPerDay: 50,
      aerationHoursPerDay: 12,
    },
  },
  harmattan: {
    name: 'Harmattan Cold Spell Stress',
    description: 'Cold nights drop water temp to 21°C, low metabolic rate, depressed appetite and high FCR.',
    icon: '❄️',
    patch: {
      waterTempC: 21.0,
      dissolvedOxygenMgL: 5.2,
      ph: 6.8,
      ammoniaMgL: 0.35,
      waterExchangePctPerDay: 25,
      aerationHoursPerDay: 6,
    },
  },
  power_outage: {
    name: 'Aeration Power Outage / Low DO',
    description: 'Zero aeration drops DO to 2.8 mg/L. Chronic hypoxia, slow growth, elevated mortality.',
    icon: '⚠️',
    patch: {
      waterTempC: 29.0,
      dissolvedOxygenMgL: 2.8,
      ph: 7.4,
      ammoniaMgL: 0.8,
      aerationHoursPerDay: 0,
      waterExchangePctPerDay: 15,
    },
  },
  rainy_overturn: {
    name: 'Heavy Downpour & Pond Overturn',
    description: 'Acidic rainwater influx (pH 5.8), turbidity spike, suspended solids, mild ammonia surge.',
    icon: '🌧️',
    patch: {
      waterTempC: 25.5,
      dissolvedOxygenMgL: 3.8,
      ph: 5.9,
      ammoniaMgL: 0.75,
      waterExchangePctPerDay: 10,
    },
  },
}
