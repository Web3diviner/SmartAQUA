/**
 * One turn in the chat transcript.
 *
 * Renders the full structured response, not just the answer text: sources,
 * confidence, risk, missing data, escalation, and safety warnings all appear
 * with the answer (04_AQUADOC_RAG_LLM.md section 12). A bare paragraph would
 * present decision support as fact.
 */

import type { ChatResponse, RecommendationTier } from '@/schemas/aquadoc'
import { ConfidenceBadge } from '@/components/ConfidenceBadge'
import { MissingDataPanel } from '@/components/MissingDataPanel'
import { RiskBadge } from '@/components/RiskBadge'
import { SourceList } from '@/components/SourceCard'
import { renderAnswer } from '@/utils/markdown'

/** 14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 5. */
const TIER_LABELS: Record<RecommendationTier, string> = {
  tier_0_informational: 'Informational',
  tier_1_advisory: 'Advisory',
  tier_2_low_risk_operational: 'Low-risk operational',
  tier_3_high_risk: 'High risk',
}

export function UserMessage({ text }: { text: string }) {
  return (
    <div className="message message--user">
      <span className="message__role">You</span>
      <p className="message__text">{text}</p>
    </div>
  )
}

export function PendingMessage() {
  return (
    <div className="message message--assistant message--pending" aria-live="polite">
      <span className="message__role">AquaDoc</span>
      <p className="message__text">
        <span className="spinner" aria-hidden="true" />
        Retrieving sources and composing an answer…
      </p>
    </div>
  )
}

export function AssistantMessage({
  response,
  devMode,
  showSources = false,
}: {
  response: ChatResponse
  devMode: boolean
  showSources?: boolean
}) {
  // Missing-data reporting only makes sense when farm context was expected.
  const contextual = response.provenance.farm_context_supplied

  return (
    <div className="message message--assistant">
      <span className="message__role">AquaDoc</span>

      {/* Text is rendered as React elements, never as an HTML string. */}
      <div className="message__answer">{renderAnswer(response.answer)}</div>

      {devMode && (
        <div className="message__badges">
          <ConfidenceBadge
            band={response.confidence_band}
            score={response.confidence}
            showNumeric={devMode}
          />
          <RiskBadge level={response.risk_level} />
          <span className="badge badge--intent">
            <span className="badge__label">Intent</span>
            <span className="badge__value">{response.intent.replace(/_/g, ' ')}</span>
          </span>
        </div>
      )}

      {response.expert_escalation && (
        <div className="escalation" role="note">
          <h4>Expert review recommended</h4>
          <ul>
            {response.escalation_reasons.map((reason) => (
              <li key={reason}>{reason}</li>
            ))}
          </ul>
        </div>
      )}

      {devMode && response.possible_causes.length > 0 && (
        <div className="causes">
          <h4>Possible causes</h4>
          <ol className="causes__list">
            {response.possible_causes.map((cause) => (
              <li key={cause.name}>
                <span className="causes__name">{cause.name}</span>
                <span className="causes__confidence">{(cause.confidence * 100).toFixed(0)}%</span>
                {cause.explanation && <p className="causes__explanation">{cause.explanation}</p>}
                {cause.supporting_source_ids.length > 0 && (
                  <p className="causes__sources">
                    Supported by {cause.supporting_source_ids.join(', ')}
                  </p>
                )}
              </li>
            ))}
          </ol>
        </div>
      )}

      {devMode && response.recommended_actions.length > 0 && (
        <div className="actions">
          <h4>Recommended actions</h4>
          <p className="actions__note">
            These are recommendations. AquaDoc does not control the feeder — any action requiring
            approval must be approved in the Smart Aqua platform before it becomes a command.
          </p>
          <ul className="actions__list">
            {response.recommended_actions.map((action) => (
              <li key={action.action} className={`actions__item actions__item--${action.tier}`}>
                <span className="actions__text">{action.action}</span>
                <span className="actions__tier">{TIER_LABELS[action.tier]}</span>
                {action.requires_approval && (
                  <span className="actions__approval">Requires approval</span>
                )}
                <p className="actions__reason">{action.reason}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

      {devMode && <MissingDataPanel labels={response.missing_data_labels} hidden={!contextual} />}

      {devMode && response.warnings.length > 0 && (
        <div className="warnings" role="note">
          <h4>Safety notes</h4>
          <ul>
            {response.warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Deterministic findings, computed without the model. */}
      {devMode && response.rule_findings.length > 0 && (
        <details className="rule-findings">
          <summary>Deterministic rule findings ({response.rule_findings.length})</summary>
          <ul>
            {response.rule_findings.map((finding) => (
              <li key={finding.rule_id} className={`rule-findings__item--${finding.status}`}>
                <code>{finding.rule_id}</code> <strong>[{finding.status}]</strong>{' '}
                {finding.summary}
              </li>
            ))}
          </ul>
        </details>
      )}

      {(showSources || devMode) && response.sources && response.sources.length > 0 && (
        <SourceList sources={response.sources} showDetails={devMode} />
      )}
    </div>
  )
}
