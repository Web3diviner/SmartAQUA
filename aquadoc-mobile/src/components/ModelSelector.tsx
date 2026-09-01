/**
 * Groq AI Model Selector Component.
 *
 * Provides instant dynamic switching between Groq reasoning,
 * safeguard, compound, and Whisper models.
 */

import { useId } from 'react'

import { useAppState } from '@/app/providers'
import { GROQ_MODELS } from '@/constants/models'

export function ModelSelector() {
  const { selectedModel, setSelectedModel } = useAppState()
  const selectId = useId()

  const current = GROQ_MODELS.find((m) => m.id === selectedModel) ?? GROQ_MODELS[0]!

  return (
    <div className="model-selector" title={`Active model: ${current.name} (${current.id})`}>
      <label htmlFor={selectId} className="model-selector__label">
        <span className="model-selector__provider-tag">Groq</span>
      </label>

      <div className="model-selector__control">
        <select
          id={selectId}
          value={selectedModel}
          onChange={(e) => setSelectedModel(e.target.value)}
          className="model-selector__select"
          aria-label="Select AI Model"
        >
          <optgroup label="Deep Reasoning & General LLMs">
            {GROQ_MODELS.filter((m) => m.category === 'reasoning' || m.category === 'compound').map((model) => (
              <option key={model.id} value={model.id}>
                {model.name} — {model.badge}
              </option>
            ))}
          </optgroup>

          <optgroup label="Safeguard & Safety Shield">
            {GROQ_MODELS.filter((m) => m.category === 'safeguard').map((model) => (
              <option key={model.id} value={model.id}>
                {model.name} — {model.badge}
              </option>
            ))}
          </optgroup>

          <optgroup label="Voice & Audio Transcription (Whisper)">
            {GROQ_MODELS.filter((m) => m.category === 'audio').map((model) => (
              <option key={model.id} value={model.id}>
                {model.name} — {model.badge}
              </option>
            ))}
          </optgroup>
        </select>
      </div>

      <span className={`model-badge model-badge--${current.category}`}>
        {current.badge}
      </span>
    </div>
  )
}
