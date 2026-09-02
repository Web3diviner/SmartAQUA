/**
 * Farm context — the simulated pond state the developer supplies.
 *
 * 15_AQUADOC_FRONTEND.md section 4: "Farm Context Simulator": before the Go
 * backend supplies real pond context, developers must be able to simulate it.
 *
 * The invariant that matters: an unmeasured value is `null`, never `0`. The
 * form models every measurement as a string so an empty input stays empty
 * rather than coercing to zero — `Number('')` is `0`, which would silently
 * turn "not measured" into "measured as zero".
 */

import { z } from 'zod'

export const waterQualitySchema = z.object({
  temperature_c: z.number().nullable(),
  ph: z.number().nullable(),
  dissolved_oxygen_mg_l: z.number().nullable(),
  turbidity_ntu: z.number().nullable(),
  ammonia_mg_l: z.number().nullable(),
  nitrite_mg_l: z.number().nullable(),
  nitrate_mg_l: z.number().nullable().optional(),
  un_ionized_ammonia_mg_l: z.number().nullable().optional(),
  orp_mv: z.number().nullable().optional(),
  salinity_ppt: z.number().nullable().optional(),
  tds_ppm: z.number().nullable().optional(),
  water_level_cm: z.number().nullable().optional(),
  alkalinity_mg_l: z.number().nullable().optional(),
  hardness_mg_l: z.number().nullable().optional(),
})

export const farmContextSchema = z.object({
  farm_id: z.string().nullable(),
  pond_id: z.string().nullable(),
  farm_name: z.string().nullable(),
  pond_name: z.string().nullable(),
  species: z.string().nullable(),
  life_stage: z.string().nullable(),
  population: z.number().int().nullable(),
  average_weight_g: z.number().nullable(),
  water: waterQualitySchema,
  feeding: z.object({
    daily_ration_g: z.number().nullable(),
    last_feeding_g: z.number().nullable(),
    feeds_per_day: z.number().int().nullable(),
    feed_acceptance: z.string().nullable(),
  }),
  health: z.object({
    mortality_24h: z.number().int().nullable(),
    mortality_7d: z.number().int().nullable(),
    active_disease_case: z.boolean().nullable(),
    reported_symptoms: z.array(z.string()),
  }),
})

export type WaterQuality = z.infer<typeof waterQualitySchema>
export type FarmContext = z.infer<typeof farmContextSchema>

/** The form's own shape: every numeric field is a string while being edited. */
export interface FarmContextForm {
  farm_name: string
  pond_name: string
  species: string
  life_stage: string
  population: string
  average_weight_g: string
  temperature_c: string
  ph: string
  dissolved_oxygen_mg_l: string
  turbidity_ntu: string
  ammonia_mg_l: string
  nitrite_mg_l: string
  nitrate_mg_l?: string
  un_ionized_ammonia_mg_l?: string
  orp_mv?: string
  salinity_ppt?: string
  tds_ppm?: string
  water_level_cm?: string
  alkalinity_mg_l?: string
  hardness_mg_l?: string
  daily_ration_g: string
  last_feeding_g: string
  feeds_per_day: string
  feed_acceptance: string
  mortality_24h: string
  reported_symptoms: string
}

/**
 * Defaults reflecting what this deployment actually measures today.
 */
export const DEFAULT_FARM_CONTEXT_FORM: FarmContextForm = {
  farm_name: 'Demo Farm',
  pond_name: 'Concrete Tank 1',
  species: 'Clarias gariepinus',
  life_stage: 'grow_out',
  population: '500',
  average_weight_g: '250',
  temperature_c: '29.4',
  ph: '',
  dissolved_oxygen_mg_l: '',
  turbidity_ntu: '',
  ammonia_mg_l: '',
  nitrite_mg_l: '',
  nitrate_mg_l: '',
  un_ionized_ammonia_mg_l: '',
  orp_mv: '',
  salinity_ppt: '',
  tds_ppm: '',
  water_level_cm: '',
  alkalinity_mg_l: '',
  hardness_mg_l: '',
  daily_ration_g: '3750',
  last_feeding_g: '1800',
  feeds_per_day: '2',
  feed_acceptance: '',
  mortality_24h: '0',
  reported_symptoms: '',
}

export const EMPTY_FARM_CONTEXT_FORM: FarmContextForm = {
  farm_name: '',
  pond_name: '',
  species: '',
  life_stage: '',
  population: '',
  average_weight_g: '',
  temperature_c: '',
  ph: '',
  dissolved_oxygen_mg_l: '',
  turbidity_ntu: '',
  ammonia_mg_l: '',
  nitrite_mg_l: '',
  nitrate_mg_l: '',
  un_ionized_ammonia_mg_l: '',
  orp_mv: '',
  salinity_ppt: '',
  tds_ppm: '',
  water_level_cm: '',
  alkalinity_mg_l: '',
  hardness_mg_l: '',
  daily_ration_g: '',
  last_feeding_g: '',
  feeds_per_day: '',
  feed_acceptance: '',
  mortality_24h: '',
  reported_symptoms: '',
}

/**
 * Parse an optional numeric form field.
 */
export function parseOptionalNumber(raw: string | undefined): number | null {
  if (!raw) return null
  const trimmed = raw.trim()
  if (trimmed === '') return null
  const value = Number(trimmed)
  return Number.isFinite(value) ? value : null
}

function parseOptionalInt(raw: string | undefined): number | null {
  const value = parseOptionalNumber(raw)
  return value === null ? null : Math.trunc(value)
}

function parseOptionalString(raw: string | undefined): string | null {
  if (!raw) return null
  const trimmed = raw.trim()
  return trimmed === '' ? null : trimmed
}

/** Convert the form into the wire payload the service expects. */
export function formToFarmContext(form: FarmContextForm): FarmContext {
  return {
    farm_id: null,
    pond_id: null,
    farm_name: parseOptionalString(form.farm_name),
    pond_name: parseOptionalString(form.pond_name),
    species: parseOptionalString(form.species),
    life_stage: parseOptionalString(form.life_stage),
    population: parseOptionalInt(form.population),
    average_weight_g: parseOptionalNumber(form.average_weight_g),
    water: {
      temperature_c: parseOptionalNumber(form.temperature_c),
      ph: parseOptionalNumber(form.ph),
      dissolved_oxygen_mg_l: parseOptionalNumber(form.dissolved_oxygen_mg_l),
      turbidity_ntu: parseOptionalNumber(form.turbidity_ntu),
      ammonia_mg_l: parseOptionalNumber(form.ammonia_mg_l),
      nitrite_mg_l: parseOptionalNumber(form.nitrite_mg_l),
      nitrate_mg_l: parseOptionalNumber(form.nitrate_mg_l),
      un_ionized_ammonia_mg_l: parseOptionalNumber(form.un_ionized_ammonia_mg_l),
      orp_mv: parseOptionalNumber(form.orp_mv),
      salinity_ppt: parseOptionalNumber(form.salinity_ppt),
      tds_ppm: parseOptionalNumber(form.tds_ppm),
      water_level_cm: parseOptionalNumber(form.water_level_cm),
      alkalinity_mg_l: parseOptionalNumber(form.alkalinity_mg_l),
      hardness_mg_l: parseOptionalNumber(form.hardness_mg_l),
    },
    feeding: {
      daily_ration_g: parseOptionalNumber(form.daily_ration_g),
      last_feeding_g: parseOptionalNumber(form.last_feeding_g),
      feeds_per_day: parseOptionalInt(form.feeds_per_day),
      feed_acceptance: parseOptionalString(form.feed_acceptance),
    },
    health: {
      mortality_24h: parseOptionalInt(form.mortality_24h),
      mortality_7d: null,
      active_disease_case: null,
      reported_symptoms: form.reported_symptoms
        .split(',')
        .map((symptom) => symptom.trim())
        .filter((symptom) => symptom.length > 0),
    },
  }
}

/** Which water measurements the simulator is currently leaving unknown. */
export function unmeasuredParameters(form: FarmContextForm): string[] {
  const labels: Record<keyof WaterQuality, string> = {
    temperature_c: 'Temperature',
    ph: 'pH',
    dissolved_oxygen_mg_l: 'Dissolved Oxygen',
    turbidity_ntu: 'Turbidity',
    ammonia_mg_l: 'Total Ammonia (TAN)',
    nitrite_mg_l: 'Nitrite (NO2)',
    nitrate_mg_l: 'Nitrate (NO3)',
    un_ionized_ammonia_mg_l: 'Un-ionized Ammonia (NH3)',
    orp_mv: 'Redox / ORP',
    salinity_ppt: 'Salinity',
    tds_ppm: 'Total Dissolved Solids',
    water_level_cm: 'Water Level',
    alkalinity_mg_l: 'Alkalinity',
    hardness_mg_l: 'Hardness',
  }
  const water = formToFarmContext(form).water
  return (Object.keys(labels) as (keyof WaterQuality)[])
    .filter((key) => water[key] === null || water[key] === undefined)
    .map((key) => labels[key])
}
