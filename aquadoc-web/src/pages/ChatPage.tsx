/**
 * Chat page — the primary AquaDoc testing interface.
 *
 * 15_AQUADOC_FRONTEND.md section 4 requires: send question, loading state,
 * retry, source cards, confidence display, risk display, missing-data display,
 * conversation reset, and a developer details toggle. Section 5 adds the
 * context selector (General Aquaculture / Simulated Pond).
 */

import { useCallback, useRef, useState } from 'react'

import { sendChat } from '@/api/chat'
import { useAppState } from '@/app/providers'
import { AssistantMessage, PendingMessage, UserMessage } from '@/components/ChatMessage'
import { ErrorPanel } from '@/components/ErrorPanel'
import { RetrievalInspector } from '@/components/RetrievalInspector'
import type { ChatResponse } from '@/schemas/aquadoc'
import { formToFarmContext } from '@/schemas/farmContext'

interface Turn {
  id: string
  question: string
  response: ChatResponse | null
  error: unknown
}

export function ChatPage() {
  const { config, chatMode, setChatMode, farmForm, devMode, debugAvailable, setDevMode } =
    useAppState()

  const [turns, setTurns] = useState<Turn[]>([])
  const [question, setQuestion] = useState('')
  const [pending, setPending] = useState(false)
  const [conversationId, setConversationId] = useState<string | null>(null)
  const inFlight = useRef<AbortController | null>(null)

  const ask = useCallback(
    async (text: string, replaceTurnId?: string) => {
      const trimmed = text.trim()
      if (!trimmed || pending) return

      inFlight.current?.abort()
      const controller = new AbortController()
      inFlight.current = controller

      const turnId = replaceTurnId ?? `turn-${Date.now()}`
      setPending(true)
      setTurns((current) =>
        replaceTurnId
          ? current.map((turn) =>
              turn.id === replaceTurnId ? { ...turn, response: null, error: null } : turn,
            )
          : [...current, { id: turnId, question: trimmed, response: null, error: null }],
      )

      try {
        const response = await sendChat(config, {
          userId: 'dev-user',
          question: trimmed,
          conversationId,
          // General mode sends no context at all — an empty object would read
          // as a pond where everything happens to be unmeasured.
          farmContext: chatMode === 'simulated_pond' ? formToFarmContext(farmForm) : null,
          signal: controller.signal,
        })
        setConversationId(response.conversation_id)
        setTurns((current) =>
          current.map((turn) => (turn.id === turnId ? { ...turn, response } : turn)),
        )
      } catch (error) {
        setTurns((current) =>
          current.map((turn) => (turn.id === turnId ? { ...turn, error } : turn)),
        )
      } finally {
        setPending(false)
        inFlight.current = null
      }
    },
    [chatMode, config, conversationId, farmForm, pending],
  )

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    void ask(question)
    setQuestion('')
  }

  const resetConversation = () => {
    inFlight.current?.abort()
    setTurns([])
    setConversationId(null)
    setPending(false)
  }

  return (
    <div className="chat-page">
      <section className="chat-page__intro" aria-labelledby="chat-title">
        <div>
          <span className="page-eyebrow">AquaDoc intelligence</span>
          <h2 id="chat-title">Good decisions start with<br /><em>better questions.</em></h2>
          <p>Your intelligent aquaculture consultant, grounded in approved knowledge and the conditions on your farm.</p>
        </div>
        <div className="chat-page__principle" aria-label="AquaDoc process">
          <span>Ask</span><i aria-hidden="true" /><span>Understand</span><i aria-hidden="true" /><span>Act</span>
        </div>
      </section>

      <header className="chat-page__header">
        <div className="chat-page__context">
          <label htmlFor="chat-mode">Context</label>
          <select
            id="chat-mode"
            value={chatMode}
            onChange={(event) => setChatMode(event.target.value as typeof chatMode)}
          >
            <option value="general">General Aquaculture</option>
            <option value="simulated_pond">Simulated Pond</option>
          </select>
          <span className="chat-page__context-note">
            {chatMode === 'general'
              ? 'Question → RAG → LLM'
              : 'Question + Farm Context + RAG + Rules → LLM'}
          </span>
        </div>

        <div className="chat-page__controls">
          {debugAvailable && (
            <label className="toggle">
              <input
                type="checkbox"
                checked={devMode}
                onChange={(event) => setDevMode(event.target.checked)}
              />
              Developer details
            </label>
          )}
          <button
            type="button"
            className="button button--ghost"
            onClick={resetConversation}
            disabled={turns.length === 0}
          >
            Reset conversation
          </button>
        </div>
      </header>

      <div className="chat-page__transcript">
        {turns.length === 0 && (
          <div className="chat-page__empty">
            <div className="chat-page__empty-heading">
              <span className="ai-orb" aria-hidden="true"><span /></span>
              <div>
                <span className="page-eyebrow">Ready when you are</span>
                <h3>How can I help with your farm today?</h3>
              </div>
            </div>
            <div className="chat-page__suggestions" aria-label="Suggested questions">
              {[
                'Why are my fish not eating?',
                'Is this temperature suitable?',
                "Calculate today's feeding rate",
                'What could cause fish to swim slowly?',
              ].map((suggestion) => (
                <button key={suggestion} type="button" onClick={() => setQuestion(suggestion)}>
                  <span>{suggestion}</span><span aria-hidden="true">↗</span>
                </button>
              ))}
            </div>
            <p className="chat-page__empty-note">
              Answers are grounded in approved knowledge. AquaDoc clearly marks missing farm data instead of making assumptions.
            </p>
          </div>
        )}

        {turns.map((turn) => (
          <section key={turn.id} className="chat-page__turn">
            <UserMessage text={turn.question} />

            {turn.response === null && turn.error === null && <PendingMessage />}

            {turn.error !== null && (
              <ErrorPanel
                error={turn.error}
                showDetails={devMode}
                onRetry={() => void ask(turn.question, turn.id)}
              />
            )}

            {turn.response && (
              <>
                <AssistantMessage response={turn.response} devMode={devMode} />
                {devMode && <RetrievalInspector response={turn.response} />}
              </>
            )}
          </section>
        ))}
      </div>

      <form className="chat-page__composer" onSubmit={handleSubmit}>
        <label className="visually-hidden" htmlFor="question">
          Ask AquaDoc
        </label>
        <textarea
          id="question"
          className="chat-page__input"
          value={question}
          onChange={(event) => setQuestion(event.target.value)}
          onKeyDown={(event) => {
            // Enter sends; Shift+Enter is a newline.
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault()
              handleSubmit(event)
            }
          }}
          placeholder="Ask AquaDoc about feeding, water quality, fish health…"
          rows={2}
          disabled={pending}
        />
        <button type="submit" className="button button--primary" disabled={pending || !question.trim()}>
          {pending ? 'Thinking…' : 'Ask AquaDoc'}
        </button>
      </form>
    </div>
  )
}
