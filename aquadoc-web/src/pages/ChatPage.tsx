/**
 * Chat page — the primary AquaDoc testing interface.
 *
 * 15_AQUADOC_FRONTEND.md section 4 requires: send question, loading state,
 * retry, source cards, confidence display, risk display, missing-data display,
 * conversation reset, and a developer details toggle. Section 5 adds the
 * context selector (General Aquaculture / Simulated Pond).
 */

import { useCallback, useRef, useState } from 'react'
import { sendChat, transcribeAudio, punctuateText } from '@/api/chat'
import { useAppState } from '@/app/providers'
import { AssistantMessage, PendingMessage, UserMessage } from '@/components/ChatMessage'
import { ErrorPanel } from '@/components/ErrorPanel'
import { RetrievalInspector } from '@/components/RetrievalInspector'
import { GROQ_MODELS } from '@/constants/models'
import type { ChatResponse } from '@/schemas/aquadoc'
import { formToFarmContext } from '@/schemas/farmContext'
import { formatSpokenText } from '@/utils/speechPunctuation'

interface Turn {
  id: string
  question: string
  response: ChatResponse | null
  error: unknown
}

function isSourceRequested(text: string): boolean {
  const lower = text.toLowerCase()
  return (
    lower.includes('source') ||
    lower.includes('reference') ||
    lower.includes('citation') ||
    lower.includes('cite') ||
    lower.includes('where did you get') ||
    lower.includes('where is this from') ||
    lower.includes('manual') ||
    lower.includes('research paper')
  )
}

export function ChatPage() {
  const {
    config,
    chatMode,
    setChatMode,
    farmForm,
    devMode,
    debugAvailable,
    setDevMode,
    selectedModel,
    user,
    isAuthenticated,
    openAuthModal,
  } = useAppState()

  const [turns, setTurns] = useState<Turn[]>([])
  const [question, setQuestion] = useState('')
  const [pending, setPending] = useState(false)
  const [conversationId, setConversationId] = useState<string | null>(null)
  const [recording, setRecording] = useState(false)
  const [transcribing, setTranscribing] = useState(false)
  const inFlight = useRef<AbortController | null>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const speechRecognizerRef = useRef<any>(null)
  const audioChunksRef = useRef<Blob[]>([])

  const currentModelInfo = GROQ_MODELS.find((m) => m.id === selectedModel) ?? GROQ_MODELS[0]!

  const startVoiceRecording = async () => {
    // 1. Try Browser Native SpeechRecognition for instant, real-time speech-to-text
    const SpeechRecognitionClass =
      (window as unknown as { SpeechRecognition?: any }).SpeechRecognition ||
      (window as unknown as { webkitSpeechRecognition?: any }).webkitSpeechRecognition

    if (SpeechRecognitionClass) {
      try {
        const recognition = new SpeechRecognitionClass()
        recognition.continuous = true
        recognition.interimResults = true
        recognition.lang = 'en-US'

        const startingText = question ? `${question.trim()} ` : ''

        recognition.onresult = (event: any) => {
          let liveTranscript = ''
          for (let i = 0; i < event.results.length; i++) {
            liveTranscript += event.results[i][0].transcript
          }
          // Live display raw transcript as user speaks
          setQuestion(startingText + liveTranscript)
        }

        recognition.onerror = (event: any) => {
          console.warn('Speech recognition notice:', event.error)
          if (event.error === 'not-allowed') {
            alert('Microphone access was denied. Please allow microphone permissions in your browser.')
            setRecording(false)
          }
        }

        recognition.onend = async () => {
          setQuestion((prev) => {
            const formatted = formatSpokenText(prev)
            // Asynchronously enhance via AI punctuation restorer if connected
            void punctuateText(config, formatted).then((aiText) => {
              if (aiText && aiText !== formatted) {
                setQuestion(aiText)
              }
            })
            return formatted
          })
          setRecording(false)
        }

        recognition.start()
        speechRecognizerRef.current = recognition
        setRecording(true)
        return
      } catch (e) {
        console.warn('Native SpeechRecognition unavailable, falling back to Whisper audio recording:', e)
      }
    }

    // 2. Fallback: Record audio and send to Groq Whisper endpoint
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      audioChunksRef.current = []
      const recorder = new MediaRecorder(stream)
      mediaRecorderRef.current = recorder

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) {
          audioChunksRef.current.push(e.data)
        }
      }

      recorder.onstop = async () => {
        stream.getTracks().forEach((track) => track.stop())
        const audioBlob = new Blob(audioChunksRef.current, { type: 'audio/wav' })
        setTranscribing(true)
        try {
          const res = await transcribeAudio(config, audioBlob, 'whisper-large-v3-turbo')
          if (res.text) {
            let formatted = formatSpokenText(res.text)
            try {
              const aiPunct = await punctuateText(config, formatted)
              if (aiPunct) formatted = aiPunct
            } catch {
              // Ignore AI punctuate error and use formatted
            }
            setQuestion((prev) => (prev ? `${prev.trim()} ${formatted}` : formatted))
          }
        } catch (err: any) {
          console.error('Voice transcription error:', err)
          alert(err.message || 'Audio transcription failed. Please check your GROQ_API_KEY in aquadoc/.env')
        } finally {
          setTranscribing(false)
        }
      }

      recorder.start()
      setRecording(true)
    } catch (err) {
      console.error('Failed to access microphone:', err)
      alert('Microphone access is required for voice queries. Please allow microphone access in your browser settings.')
    }
  }

  const stopVoiceRecording = () => {
    if (speechRecognizerRef.current) {
      try {
        speechRecognizerRef.current.stop()
      } catch {
        // Ignored if already stopped
      }
      speechRecognizerRef.current = null
    }

    if (mediaRecorderRef.current && recording) {
      mediaRecorderRef.current.stop()
    }
    setQuestion((prev) => {
      const formatted = formatSpokenText(prev)
      void punctuateText(config, formatted).then((aiText) => {
        if (aiText && aiText !== formatted) {
          setQuestion(aiText)
        }
      })
      return formatted
    })
    setRecording(false)
  }

  const ask = useCallback(
    async (text: string, replaceTurnId?: string) => {
      const trimmed = text.trim()
      if (!trimmed || pending) return

      // Prompt sign up / login if unauthenticated
      if (!isAuthenticated) {
        openAuthModal()
        return
      }

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
          userId: user?.id || 'farmer-session',
          question: trimmed,
          conversationId,
          farmContext: chatMode === 'simulated_pond' ? formToFarmContext(farmForm) : null,
          model: selectedModel,
          signal: controller.signal,
        })
        setConversationId(response.conversation_id)
        setTurns((current) =>
          current.map((turn) => (turn.id === turnId ? { ...turn, response } : turn)),
        )
      } catch (err) {
        if (controller.signal.aborted) return
        const message = err instanceof Error ? err.message : 'Failed to consult AquaDoc'
        setTurns((current) =>
          current.map((turn) =>
            turn.id === turnId ? { ...turn, error: message } : turn,
          ),
        )
      } finally {
        setPending(false)
        inFlight.current = null
      }
    },
    [config, pending, conversationId, chatMode, farmForm, selectedModel, isAuthenticated, openAuthModal, user],
  )

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    if (!isAuthenticated) {
      openAuthModal()
      return
    }
    const current = question
    setQuestion('')
    void ask(current)
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
            {currentModelInfo.name} · {currentModelInfo.badge}
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
                <span className="page-eyebrow">
                  {isAuthenticated ? `Welcome, ${user?.name}` : 'Sign In Required for Consultation'}
                </span>
                <h3>
                  {isAuthenticated
                    ? 'How can I assist your farm today?'
                    : 'Sign in or register to get started with AquaDoc AI'}
                </h3>
              </div>
            </div>

            {!isAuthenticated && (
              <div className="auth-prompt-banner">
                <p>
                  To receive clinical advice, tailored feed calculations, and disease prescriptions, please sign in with your <strong>Google Account (Gmail)</strong> or register your farm details.
                </p>
                <button
                  type="button"
                  className="button button--primary"
                  onClick={openAuthModal}
                >
                  🚀 Sign In / Register Farm Account
                </button>
              </div>
            )}

            <div className="chat-page__suggestions" aria-label="Suggested questions">
              {[
                'Why are my fish not eating?',
                'Is this temperature suitable?',
                "Calculate today's feeding rate",
                'What could cause fish to swim slowly?',
              ].map((suggestion) => (
                <button
                  key={suggestion}
                  type="button"
                  onClick={() => {
                    if (!isAuthenticated) {
                      openAuthModal()
                      return
                    }
                    setQuestion(suggestion)
                    void ask(suggestion)
                  }}
                  title="Click to ask this question"
                >
                  <span>{suggestion}</span>
                  <span aria-hidden="true">↗</span>
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
                <AssistantMessage
                  response={turn.response}
                  devMode={devMode}
                  showSources={isSourceRequested(turn.question)}
                />
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
        
        {recording && (
          <div className="voice-recording-banner">
            <span className="recording-pulse-dot" aria-hidden="true" />
            <span>Listening... Speak your aquaculture question (Groq Whisper)</span>
            <button
              type="button"
              className="button button--ghost button--sm"
              onClick={stopVoiceRecording}
            >
              Done
            </button>
          </div>
        )}

        <div className="composer-input-row">
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
            rows={2}
            disabled={pending || recording || transcribing}
          />

          <div className="composer-actions">
            <button
              type="button"
              className={`button-mic ${recording ? 'button-mic--recording' : ''}`}
              onClick={recording ? stopVoiceRecording : startVoiceRecording}
              disabled={pending || transcribing}
              title={recording ? 'Stop recording' : 'Speak question with Whisper'}
              aria-label={recording ? 'Stop recording' : 'Voice input'}
            >
              {transcribing ? (
                <span className="spinner-sm" />
              ) : (
                <svg viewBox="0 0 24 24" className="mic-icon" aria-hidden="true">
                  <path
                    d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"
                    fill="currentColor"
                  />
                  <path
                    d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"
                    fill="currentColor"
                  />
                </svg>
              )}
            </button>

            <button
              type="submit"
              className="button button--primary"
              disabled={pending || recording || transcribing || !question.trim()}
            >
              {pending ? 'Thinking…' : 'Ask AquaDoc'}
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}
