/**
 * Frontend unit tests.
 *
 * 15_AQUADOC_FRONTEND.md section 18 lists the unit-test targets: response
 * rendering, source cards, confidence labels, missing-data display, and schema
 * validation. The missing-data and XSS cases are the load-bearing ones — both
 * are silent failures if they regress.
 */

import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ConfidenceBadge } from '@/components/ConfidenceBadge'
import { MeasurementValue, MissingDataPanel } from '@/components/MissingDataPanel'
import { SourceCard } from '@/components/SourceCard'
import { chatResponseSchema } from '@/schemas/aquadoc'
import {
  DEFAULT_FARM_CONTEXT_FORM,
  EMPTY_FARM_CONTEXT_FORM,
  formToFarmContext,
  parseOptionalNumber,
  unmeasuredParameters,
} from '@/schemas/farmContext'
import { renderAnswer } from '@/utils/markdown'

describe('missing-data policy', () => {
  it('treats a blank field as unknown, not zero', () => {
    // Number('') === 0 — the coercion this guards against.
    expect(parseOptionalNumber('')).toBeNull()
    expect(parseOptionalNumber('   ')).toBeNull()
    expect(parseOptionalNumber('not a number')).toBeNull()
  })

  it('preserves a genuine zero reading', () => {
    // 0.0 mg/L dissolved oxygen is a pond-killing event, not a missing value.
    expect(parseOptionalNumber('0')).toBe(0)
    expect(parseOptionalNumber('0.0')).toBe(0)
  })

  it('sends null for every unmeasured parameter', () => {
    const context = formToFarmContext(DEFAULT_FARM_CONTEXT_FORM)

    expect(context.water.temperature_c).toBe(29.4)
    expect(context.water.ph).toBeNull()
    expect(context.water.dissolved_oxygen_mg_l).toBeNull()
    expect(context.water.turbidity_ntu).toBeNull()
  })

  it('defaults reflect the sensors actually installed', () => {
    // Temperature is live; pH, DO and turbidity are not installed yet.
    expect(unmeasuredParameters(DEFAULT_FARM_CONTEXT_FORM)).toEqual([
      'pH',
      'Dissolved Oxygen',
      'Turbidity',
      'Ammonia',
      'Nitrite',
    ])
  })

  it('reports every parameter unmeasured for an empty form', () => {
    expect(unmeasuredParameters(EMPTY_FARM_CONTEXT_FORM)).toHaveLength(6)
  })

  it('renders an unavailable measurement as text, never as 0', () => {
    render(<MeasurementValue value={null} unit="mg/L" />)

    expect(screen.getByText('Not available')).toBeInTheDocument()
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  it('renders a zero measurement as 0, not as unavailable', () => {
    render(<MeasurementValue value={0} unit="mg/L" />)

    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.queryByText('Not available')).not.toBeInTheDocument()
  })

  it('lists unavailable parameters with an explanation', () => {
    render(<MissingDataPanel labels={['pH', 'Dissolved Oxygen']} />)

    expect(screen.getByText('Data not currently available')).toBeInTheDocument()
    expect(screen.getByText('pH')).toBeInTheDocument()
    expect(screen.getByText(/unknown — not normal/i)).toBeInTheDocument()
  })

  it('is suppressed for general questions', () => {
    const { container } = render(<MissingDataPanel labels={['pH']} hidden />)

    expect(container).toBeEmptyDOMElement()
  })
})

describe('confidence display', () => {
  it('shows a band and hides the number from farmers', () => {
    render(<ConfidenceBadge band="moderate" score={0.62} />)

    expect(screen.getByText('Moderate')).toBeInTheDocument()
    expect(screen.queryByText(/0\.62/)).not.toBeInTheDocument()
  })

  it('shows the number in developer mode', () => {
    render(<ConfidenceBadge band="high" score={0.81} showNumeric />)

    expect(screen.getByText('High')).toBeInTheDocument()
    expect(screen.getByText(/0\.81/)).toBeInTheDocument()
  })
})

describe('source cards', () => {
  const source = {
    chunk_id: 'S1',
    document_id: 'doc-1',
    title: 'FAO Aquaculture Manual',
    source: 'FAO',
    author: null,
    year: 2022,
    page: 48,
    section: null,
    evidence_level: 'A' as const,
    excerpt: 'Feed conversion ratio compares feed input to weight gain.',
    score: 0.87,
    chunk_text: 'The full retrieved chunk text.',
  }

  it('shows title and page', () => {
    render(<SourceCard source={source} />)

    expect(screen.getByText('FAO Aquaculture Manual')).toBeInTheDocument()
    expect(screen.getByText('48')).toBeInTheDocument()
  })

  it('labels an unpaginated document rather than showing a bogus page', () => {
    render(<SourceCard source={{ ...source, page: null }} />)

    expect(screen.getByText('Not paginated')).toBeInTheDocument()
  })

  it('hides the retrieved chunk and score outside developer mode', () => {
    render(<SourceCard source={source} />)

    expect(screen.queryByText(/full retrieved chunk/i)).not.toBeInTheDocument()
    expect(screen.queryByText('0.870')).not.toBeInTheDocument()
  })

  it('offers the retrieved chunk in developer mode', () => {
    render(<SourceCard source={source} showDetails />)

    expect(screen.getByRole('button', { name: /full retrieved chunk/i })).toBeInTheDocument()
    expect(screen.getByText('0.870')).toBeInTheDocument()
  })
})

describe('answer rendering', () => {
  it('never interprets model output as HTML', () => {
    // 15_AQUADOC_FRONTEND.md section 10 — the text is parsed into React
    // elements, so markup arrives as literal characters.
    const malicious = 'Before <script>alert("xss")</script> after <img src=x onerror=alert(1)>'
    const { container } = render(<div>{renderAnswer(malicious)}</div>)

    expect(container.querySelector('script')).toBeNull()
    expect(container.querySelector('img')).toBeNull()
    expect(container.textContent).toContain('<script>')
  })

  it('renders bullet lists as list elements', () => {
    const { container } = render(<div>{renderAnswer('Causes:\n\n- Low oxygen\n- Cold water')}</div>)

    expect(container.querySelectorAll('li')).toHaveLength(2)
  })

  it('renders numbered lists as ordered lists', () => {
    const { container } = render(<div>{renderAnswer('1. First step\n2. Second step')}</div>)

    expect(container.querySelector('ol')).not.toBeNull()
    expect(container.querySelectorAll('li')).toHaveLength(2)
  })

  it('renders bold and code inline', () => {
    const { container } = render(<div>{renderAnswer('Keep **oxygen** above `5 mg/L`.')}</div>)

    expect(container.querySelector('strong')?.textContent).toBe('oxygen')
    expect(container.querySelector('code')?.textContent).toBe('5 mg/L')
  })
})

describe('response validation', () => {
  const valid = {
    request_id: 'REQ-1',
    conversation_id: 'conv-1',
    answer: 'Feed conversion ratio compares feed input to weight gain.',
    intent: 'general_aquaculture',
    risk_level: 'informational',
    confidence: 0.72,
    confidence_band: 'moderate',
    provenance: {
      prompt_version: 'general_v1@v1',
      llm_model: 'claude-opus-5',
      llm_provider: 'anthropic',
      embedding_model: 'voyage-3',
      embedding_provider: 'voyage',
      rules_version: 'water_quality/v1',
      generated_at: '2026-08-08T10:00:00Z',
    },
  }

  it('accepts a well-formed response', () => {
    const parsed = chatResponseSchema.safeParse(valid)

    expect(parsed.success).toBe(true)
  })

  it('rejects a confidence outside 0-1', () => {
    expect(chatResponseSchema.safeParse({ ...valid, confidence: 1.5 }).success).toBe(false)
  })

  it('rejects an unknown risk level rather than rendering it', () => {
    expect(chatResponseSchema.safeParse({ ...valid, risk_level: 'catastrophic' }).success).toBe(
      false,
    )
  })

  it('rejects a response missing provenance', () => {
    const { provenance: _omitted, ...withoutProvenance } = valid
    expect(chatResponseSchema.safeParse(withoutProvenance).success).toBe(false)
  })

  it('defaults optional collections so the UI never reads undefined', () => {
    const parsed = chatResponseSchema.parse(valid)

    expect(parsed.sources).toEqual([])
    expect(parsed.missing_data).toEqual([])
    expect(parsed.recommended_actions).toEqual([])
  })
})
