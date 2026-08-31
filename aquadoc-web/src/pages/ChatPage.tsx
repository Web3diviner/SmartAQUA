/**
 * Chat page — the primary AquaDoc testing interface.
 *
 * 15_AQUADOC_FRONTEND.md section 4 requires: send question, loading state,
 * retry, source cards, confidence display, risk display, missing-data display,
 * conversation reset, and continuous memory with chat session history.
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import { sendChat, transcribeAudio, punctuateText, deleteConversationApi } from '@/api/chat'
import { useAppState } from '@/app/providers'
import { AssistantMessage, PendingMessage, UserMessage } from '@/components/ChatMessage'
import { ChatHistorySidebar } from '@/components/ChatHistorySidebar'
import { ErrorPanel } from '@/components/ErrorPanel'
import { GROQ_MODELS } from '@/constants/models'
import { formToFarmContext } from '@/schemas/farmContext'
import {
  type ChatSession,
  type ChatTurn,
  createNewSession,
  loadSavedSessions,
  saveSessionsToStorage,
} from '@/state/chatHistoryStore'
import { formatSpokenText } from '@/utils/speechPunctuation'

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
    selectedModel,
    user,
    isAuthenticated,
    openAuthModal,
  } = useAppState()

  // Chat Sessions & History State
  const [sessions, setSessions] = useState<ChatSession[]>(() => loadSavedSessions())
  const [activeSessionId, setActiveSessionId] = useState<string | null>(() => {
    const saved = loadSavedSessions()
    return saved.length > 0 ? saved[0]!.id : null
  })
  const [isHistoryOpen, setIsHistoryOpen] = useState(false)

  // Current dialogue turns & input
  const [turns, setTurns] = useState<ChatTurn[]>(() => {
    const saved = loadSavedSessions()
    return saved.length > 0 ? saved[0]!.turns : []
  })
  const [question, setQuestion] = useState('')
  const [pending, setPending] = useState(false)
  const [recording, setRecording] = useState(false)
  const [transcribing, setTranscribing] = useState(false)
  const inFlight = useRef<AbortController | null>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const speechRecognizerRef = useRef<any>(null)
  const audioChunksRef = useRef<Blob[]>([])

  const currentModelInfo = GROQ_MODELS.find((m) => m.id === selectedModel) ?? GROQ_MODELS[0]!

  // Save sessions to localStorage whenever sessions change
  useEffect(() => {
    saveSessionsToStorage(sessions)
  }, [sessions])

  // Sync active session's turns into the sessions list
  const syncTurnsToSession = useCallback(
    (currentTurns: ChatTurn[], currentSessionId: string, initialQuestion?: string) => {
      setSessions((prevSessions) => {
        const existingIndex = prevSessions.findIndex((s) => s.id === currentSessionId)
        const now = new Date().toISOString()

        if (existingIndex >= 0) {
          const updated = [...prevSessions]
          const prev = updated[existingIndex]!
          updated[existingIndex] = {
            ...prev,
            updatedAt: now,
            turns: currentTurns,
            title:
              prev.title === 'New AquaDoc Consultation' && initialQuestion
                ? initialQuestion.substring(0, 50)
                : prev.title,
          }
          return updated
        } else {
          // Create new session entry
          const newSession: ChatSession = {
            id: currentSessionId,
            title: initialQuestion ? initialQuestion.substring(0, 50) : 'New AquaDoc Consultation',
            createdAt: now,
            updatedAt: now,
            turns: currentTurns,
          }
          return [newSession, ...prevSessions]
        }
      })
    },
    [],
  )

  const handleSelectSession = (session: ChatSession) => {
    inFlight.current?.abort()
    setActiveSessionId(session.id)
    setTurns(session.turns || [])
    setPending(false)
    setQuestion('')
  }

  const handleNewSession = () => {
    inFlight.current?.abort()
    const session = createNewSession()
    setSessions((prev) => [session, ...prev])
    setActiveSessionId(session.id)
    setTurns([])
    setPending(false)
    setQuestion('')
  }

  const handleDeleteSession = async (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    const toDelete = sessions.find((s) => s.id === sessionId)
    const title = toDelete?.title || 'this consultation'
    if (!window.confirm(`Are you sure you want to delete "${title}"?`)) return

    void deleteConversationApi(config, sessionId)

    setSessions((prev) => {
      const remaining = prev.filter((s) => s.id !== sessionId)
      if (activeSessionId === sessionId) {
        if (remaining.length > 0) {
          setActiveSessionId(remaining[0]!.id)
          setTurns(remaining[0]!.turns)
        } else {
          const fresh = createNewSession()
          setActiveSessionId(fresh.id)
          setTurns([])
          return [fresh]
        }
      }
      return remaining
    })
  }

  const startVoiceRecording = async () => {
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
            void punctuateText(config, formatted).then((punctuated) => {
              if (punctuated && punctuated !== formatted) {
                setQuestion(punctuated)
              }
            })
            return formatted
          })
          setRecording(false)
        }

        speechRecognizerRef.current = recognition
        recognition.start()
        setRecording(true)
        return
      } catch (e) {
        console.warn('Native speech recognition fallback to Groq Whisper:', e)
      }
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      audioChunksRef.current = []
      const recorder = new MediaRecorder(stream)

      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          audioChunksRef.current.push(event.data)
        }
      }

      recorder.onstop = async () => {
        const audioBlob = new Blob(audioChunksRef.current, { type: 'audio/wav' })
        stream.getTracks().forEach((track) => track.stop())
        setTranscribing(true)
        try {
          const result = await transcribeAudio(config, audioBlob)
          if (result.text) {
            const punctuated = await punctuateText(config, result.text)
            setQuestion((prev) => (prev ? `${prev} ${punctuated}` : punctuated))
          }
        } catch (err) {
          console.error('Audio transcription error:', err)
          alert('Could not transcribe audio. Please check your microphone or type your question.')
        } finally {
          setTranscribing(false)
        }
      }

      mediaRecorderRef.current = recorder
      recorder.start()
      setRecording(true)
    } catch (err) {
      console.error('Microphone access error:', err)
      alert('Microphone access is required to use voice input. Please check your browser permissions.')
    }
  }

  const stopVoiceRecording = () => {
    if (speechRecognizerRef.current) {
      try {
        speechRecognizerRef.current.stop()
      } catch {}
      speechRecognizerRef.current = null
    }
    if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
      mediaRecorderRef.current.stop()
    }
    setRecording(false)
  }

  const toggleRecording = () => {
    if (recording) {
      stopVoiceRecording()
    } else {
      void startVoiceRecording()
    }
  }

  const ask = useCallback(
    async (text: string, replaceTurnId?: string) => {
      const trimmed = text.trim()
      if (!trimmed || pending) return

      if (!isAuthenticated) {
        openAuthModal()
        return
      }

      inFlight.current?.abort()
      const controller = new AbortController()
      inFlight.current = controller

      // Guarantee a session ID exists
      let currentSessionId = activeSessionId
      if (!currentSessionId) {
        const fresh = createNewSession(trimmed)
        currentSessionId = fresh.id
        setActiveSessionId(fresh.id)
      }

      const turnId = replaceTurnId ?? `turn-${Date.now()}`
      setPending(true)

      const nextTurns: ChatTurn[] = replaceTurnId
        ? turns.map((turn) =>
            turn.id === replaceTurnId ? { ...turn, response: null, error: null } : turn,
          )
        : [...turns, { id: turnId, question: trimmed, response: null, error: null }]

      setTurns(nextTurns)
      syncTurnsToSession(nextTurns, currentSessionId, turns.length === 0 ? trimmed : undefined)

      try {
        const response = await sendChat(config, {
          userId: user?.id || 'farmer-session',
          question: trimmed,
          conversationId: currentSessionId,
          farmContext: chatMode === 'simulated_pond' ? formToFarmContext(farmForm) : null,
          model: selectedModel,
          signal: controller.signal,
        })

        const resolvedTurns = nextTurns.map((turn) =>
          turn.id === turnId ? { ...turn, response } : turn,
        )
        setTurns(resolvedTurns)
        syncTurnsToSession(resolvedTurns, currentSessionId, turns.length === 0 ? trimmed : undefined)
      } catch (err) {
        if (controller.signal.aborted) return
        const message = err instanceof Error ? err.message : 'Failed to consult AquaDoc'
        const errorTurns = nextTurns.map((turn) =>
          turn.id === turnId ? { ...turn, error: message } : turn,
        )
        setTurns(errorTurns)
        syncTurnsToSession(errorTurns, currentSessionId)
      } finally {
        setPending(false)
        inFlight.current = null
      }
    },
    [
      config,
      pending,
      activeSessionId,
      turns,
      chatMode,
      farmForm,
      selectedModel,
      isAuthenticated,
      openAuthModal,
      user,
      syncTurnsToSession,
    ],
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
    handleNewSession()
  }

  return (
    <div className="chat-layout-wrapper">
      {/* Consultation History Sidebar */}
      <ChatHistorySidebar
        isOpen={isHistoryOpen}
        onClose={() => setIsHistoryOpen(false)}
        sessions={sessions}
        activeSessionId={activeSessionId}
        onSelectSession={handleSelectSession}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
      />

      <div className="chat-page">
        <section className="chat-page__intro" aria-labelledby="chat-title">
          <div>
            <span className="page-eyebrow">AquaDoc intelligence</span>
            <h2 id="chat-title">
              Good decisions start with
              <br />
              <em>better questions.</em>
            </h2>
            <p>
              Your intelligent aquaculture consultant, grounded in approved knowledge and the
              conditions on your farm with continuous conversational memory.
            </p>
          </div>
          <div className="chat-page__principle" aria-label="AquaDoc process">
            <span>Ask</span>
            <i aria-hidden="true" />
            <span>Understand</span>
            <i aria-hidden="true" />
            <span>Act</span>
          </div>
        </section>

        <header className="chat-page__header">
          <div className="chat-page__context">
            {/* History Drawer Toggle Button */}
            <button
              type="button"
              className="chat-history-toggle-btn"
              onClick={() => setIsHistoryOpen((prev) => !prev)}
              title="View Consultation History"
            >
              <span className="btn-icon">📁</span>
              <span className="btn-text">History</span>
              {sessions.length > 0 && <span className="history-badge">{sessions.length}</span>}
            </button>

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
            <button
              type="button"
              className="button button--ghost new-consultation-btn"
              onClick={handleNewSession}
            >
              ✨ New Consultation
            </button>
            <button
              type="button"
              className="button button--ghost"
              onClick={resetConversation}
              disabled={turns.length === 0}
            >
              Clear
            </button>
          </div>
        </header>

        <div className="chat-page__transcript">
          {turns.length === 0 && (
            <div className="chat-page__empty">
              <div className="chat-page__empty-heading">
                <span className="ai-orb" aria-hidden="true">
                  <span />
                </span>
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
                    To receive clinical advice, tailored feed calculations, and disease prescriptions,
                    please sign in with your <strong>Google Account (Gmail)</strong> or register your
                    farm details.
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
                  'What is the treatment dosage of bitter leaf extract?',
                  "Calculate today's feeding rate",
                  'What could cause catfish to swim sluggishly at the surface?',
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
            </div>
          )}

          {turns.map((turn) => (
            <section key={turn.id} className="chat-page__turn">
              <UserMessage text={turn.question} />

              {turn.response === null && turn.error === null && <PendingMessage />}

              {turn.error !== null && (
                <ErrorPanel
                  error={turn.error}
                  showDetails={false}
                  onRetry={() => void ask(turn.question, turn.id)}
                />
              )}

              {turn.response !== null && (
                <AssistantMessage
                  response={turn.response}
                  devMode={false}
                  showSources={isSourceRequested(turn.question)}
                />
              )}
            </section>
          ))}
        </div>

        <form className="chat-page__input-bar" onSubmit={handleSubmit}>
          <div className="chat-page__input-container">
            <textarea
              className="chat-page__textarea"
              placeholder={
                recording
                  ? 'Listening to your question... Speak naturally'
                  : transcribing
                    ? 'Processing voice transcription...'
                    : turns.length > 0
                      ? 'Ask a follow-up question (AquaDoc remembers earlier context)...'
                      : 'Ask AquaDoc anything about your fish, water, or feeding...'
              }
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSubmit(e)
                }
              }}
              rows={1}
              disabled={pending || transcribing}
            />

            <div className="chat-page__action-buttons">
              <button
                type="button"
                className={`button-mic ${recording ? 'recording' : ''} ${transcribing ? 'transcribing' : ''}`}
                onClick={toggleRecording}
                disabled={pending || transcribing}
                title={recording ? 'Stop Recording' : 'Voice Input (Ask via Audio)'}
              >
                {recording ? (
                  <span className="recording-wave">
                    <span className="dot" />
                    <span className="dot" />
                    <span className="dot" />
                  </span>
                ) : transcribing ? (
                  <span className="spinner-icon">⏳</span>
                ) : (
                  <svg
                    width="18"
                    height="18"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
                    <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                    <line x1="12" y1="19" x2="12" y2="23" />
                    <line x1="8" y1="23" x2="16" y2="23" />
                  </svg>
                )}
              </button>

              <button
                type="submit"
                className="button-send"
                disabled={!question.trim() || pending || transcribing}
                title="Send Question (Enter)"
              >
                {pending ? (
                  <div className="button-spinner" />
                ) : (
                  <svg
                    width="18"
                    height="18"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <line x1="22" y1="2" x2="11" y2="13" />
                    <polygon points="22 2 15 22 11 13 2 9 22 2" />
                  </svg>
                )}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
