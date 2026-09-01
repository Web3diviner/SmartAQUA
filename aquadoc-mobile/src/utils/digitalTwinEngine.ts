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

export interface RecommendedTankDimensions {
  systemType: PondSystemType
  systemLabel: string
  requiredWaterVolumeM3: number
  currentWaterVolumeM3: number
  capacityUtilizationPct: number
  capacityStatus: 'underutilized' | 'optimal' | 'moderate_overcrowding' | 'severely_undersized'
  safeDensityLimitKgM3: number
  projectedHarvestDensityKgM3: number
  lengthM?: number
  widthM?: number
  heightM?: number
  lengthFt?: number
  widthFt?: number
  heightFt?: number
  diameterM?: number
  diameterFt?: number
  depthM: number
  depthFt: number
  numberOfTanksRecommended: number
  singleTankVolumeM3: number
  singleTankDimensionsLabel: string
  singleTankDimensionsFeetLabel: string
  nurseryPhaseVolumeM3: number
  growoutPhaseVolumeM3: number
  sizeToDurationRatioInsight: string
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
  recommendedTankDimensions: RecommendedTankDimensions
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

  const recommendedTankDimensions = calculateRecommendedTankDimensions(
    params.systemType,
    Number(finalBiomassKg.toFixed(1)),
    params.waterVolumeM3,
    days,
    params.initialAvgWeightG,
    Number(currentWeight.toFixed(1)),
    params.aerationHoursPerDay,
  )

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
    recommendedTankDimensions,
  }
}

/**
 * Calculates optimal physical tank dimensions and volume needed for the projected
 * fish size to duration ratio to maintain healthy carrying densities.
 */
export function calculateRecommendedTankDimensions(
  systemType: PondSystemType,
  finalBiomassKg: number,
  currentWaterVolumeM3: number,
  horizonDays: number,
  initialWeightG: number,
  finalWeightG: number,
  aerationHours: number,
): RecommendedTankDimensions {
  const systemConfig = SYSTEM_CAPACITY_LIMITS[systemType]
  const aerationBoost = aerationHours >= 10 ? 1.2 : aerationHours >= 6 ? 1.1 : 1.0
  const safeDensityLimitKgM3 = Number((systemConfig.maxDensityKgM3 * aerationBoost).toFixed(1))

  const requiredWaterVolumeM3 = Number(((finalBiomassKg / safeDensityLimitKgM3) * 1.1).toFixed(1))
  const projectedHarvestDensityKgM3 = Number((finalBiomassKg / Math.max(0.1, currentWaterVolumeM3)).toFixed(1))
  const capacityUtilizationPct = Number(((projectedHarvestDensityKgM3 / safeDensityLimitKgM3) * 100).toFixed(0))

  let capacityStatus: RecommendedTankDimensions['capacityStatus'] = 'optimal'
  if (capacityUtilizationPct < 65) {
    capacityStatus = 'underutilized'
  } else if (capacityUtilizationPct <= 105) {
    capacityStatus = 'optimal'
  } else if (capacityUtilizationPct <= 140) {
    capacityStatus = 'moderate_overcrowding'
  } else {
    capacityStatus = 'severely_undersized'
  }

  const nurseryPhaseVolumeM3 = Number((requiredWaterVolumeM3 * 0.28).toFixed(1))
  const growoutPhaseVolumeM3 = requiredWaterVolumeM3

  const M_TO_FT = 3.28084
  let depthM = 1.2
  let numberOfTanksRecommended = 1
  let singleTankVolumeM3 = requiredWaterVolumeM3

  let lengthM: number | undefined
  let widthM: number | undefined
  let heightM: number | undefined
  let lengthFt: number | undefined
  let widthFt: number | undefined
  let heightFt: number | undefined
  let diameterM: number | undefined
  let diameterFt: number | undefined
  let singleTankDimensionsLabel = ''
  let singleTankDimensionsFeetLabel = ''

  if (systemType === 'concrete') {
    depthM = 1.2
    const totalWallHeight = 1.5
    if (requiredWaterVolumeM3 > 50) {
      numberOfTanksRecommended = Math.ceil(requiredWaterVolumeM3 / 40)
      singleTankVolumeM3 = Number((requiredWaterVolumeM3 / numberOfTanksRecommended).toFixed(1))
    }
    const floorArea = singleTankVolumeM3 / depthM
    const w = Math.sqrt(floorArea / 2)
    const l = 2 * w
    widthM = Number(w.toFixed(1))
    lengthM = Number(l.toFixed(1))
    heightM = totalWallHeight
    widthFt = Number((w * M_TO_FT).toFixed(1))
    lengthFt = Number((l * M_TO_FT).toFixed(1))
    heightFt = Number((totalWallHeight * M_TO_FT).toFixed(1))

    singleTankDimensionsLabel = `${lengthM}m (L) × ${widthM}m (W) × ${heightM}m (H)`
    singleTankDimensionsFeetLabel = `${lengthFt}ft (L) × ${widthFt}ft (W) × ${heightFt}ft (H)`
  } else if (systemType === 'tarpaulin' || systemType === 'ras') {
    depthM = systemType === 'ras' ? 1.3 : 1.0
    const totalHeight = systemType === 'ras' ? 1.5 : 1.2
    const maxSingleVol = systemType === 'ras' ? 45 : 30
    if (requiredWaterVolumeM3 > maxSingleVol) {
      numberOfTanksRecommended = Math.ceil(requiredWaterVolumeM3 / maxSingleVol)
      singleTankVolumeM3 = Number((requiredWaterVolumeM3 / numberOfTanksRecommended).toFixed(1))
    }
    const floorArea = singleTankVolumeM3 / depthM
    const d = 2 * Math.sqrt(floorArea / Math.PI)
    diameterM = Number(d.toFixed(1))
    diameterFt = Number((d * M_TO_FT).toFixed(1))
    heightM = totalHeight
    heightFt = Number((totalHeight * M_TO_FT).toFixed(1))

    singleTankDimensionsLabel = `Ø ${diameterM}m Diameter × ${heightM}m Height`
    singleTankDimensionsFeetLabel = `Ø ${diameterFt}ft Diameter × ${heightFt}ft Height`
  } else {
    depthM = 1.3
    const totalHeight = 1.7
    if (requiredWaterVolumeM3 > 350) {
      numberOfTanksRecommended = Math.ceil(requiredWaterVolumeM3 / 250)
      singleTankVolumeM3 = Number((requiredWaterVolumeM3 / numberOfTanksRecommended).toFixed(1))
    }
    const floorArea = singleTankVolumeM3 / depthM
    const l = Math.sqrt(1.8 * floorArea)
    const w = floorArea / l
    widthM = Number(w.toFixed(1))
    lengthM = Number(l.toFixed(1))
    heightM = totalHeight
    widthFt = Number((w * M_TO_FT).toFixed(1))
    lengthFt = Number((l * M_TO_FT).toFixed(1))
    heightFt = Number((totalHeight * M_TO_FT).toFixed(1))

    singleTankDimensionsLabel = `${lengthM}m (L) × ${widthM}m (W) × ${heightM}m Depth (${Number(floorArea.toFixed(0))}m²)`
    singleTankDimensionsFeetLabel = `${lengthFt}ft (L) × ${widthFt}ft (W) × ${heightFt}ft Depth`
  }

  const weightGainRatio = Number((finalWeightG / Math.max(1, initialWeightG)).toFixed(1))
  const growthRateGPerDay = Number(((finalWeightG - initialWeightG) / horizonDays).toFixed(1))
  const sizeToDurationRatioInsight = `For a ${horizonDays}-day cycle achieving a ${weightGainRatio}x size expansion (${initialWeightG}g → ${finalWeightG}g at ~${growthRateGPerDay}g/day), your stock will reach ${finalBiomassKg.toLocaleString()}kg harvest biomass. This requires a minimum of ${requiredWaterVolumeM3}m³ (${(requiredWaterVolumeM3 * 1000).toLocaleString()} Liters) across ${numberOfTanksRecommended} ${systemConfig.label}${numberOfTanksRecommended > 1 ? 's' : ''} to prevent stunting and cannibalism.`

  return {
    systemType,
    systemLabel: systemConfig.label,
    requiredWaterVolumeM3,
    currentWaterVolumeM3,
    capacityUtilizationPct,
    capacityStatus,
    safeDensityLimitKgM3,
    projectedHarvestDensityKgM3,
    lengthM,
    widthM,
    heightM,
    lengthFt,
    widthFt,
    heightFt,
    diameterM,
    diameterFt,
    depthM,
    depthFt: Number((depthM * M_TO_FT).toFixed(1)),
    numberOfTanksRecommended,
    singleTankVolumeM3,
    singleTankDimensionsLabel,
    singleTankDimensionsFeetLabel,
    nurseryPhaseVolumeM3,
    growoutPhaseVolumeM3,
    sizeToDurationRatioInsight,
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
