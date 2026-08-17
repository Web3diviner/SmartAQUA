/**
 * Routing and application shell.
 *
 * Scope is the first frontend milestone (15_AQUADOC_FRONTEND.md section 22):
 * Chat, Farm Context Simulator, Sources, and the Retrieval Inspector. Knowledge
 * administration, disease testing, and evaluation come only after this is
 * reliable — Sources and the Inspector are rendered inline with each answer
 * rather than as separate routes, since they exist to be read alongside it.
 */

import { NavLink, Navigate, Route, Routes } from 'react-router-dom'

import { useAppState } from '@/app/providers'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { ChatPage } from '@/pages/ChatPage'
import { FarmSimulatorPage } from '@/pages/FarmSimulatorPage'

export function App() {
  const { config, debugAvailable } = useAppState()

  return (
    <div className="app">
      <header className="app__header">
        <div className="app__brand">
          <BrandMark />
          <div className="app__brand-copy">
            <span className="app__ecosystem">Smart Aqua</span>
            <h1>AquaDoc <span>AI</span></h1>
          </div>
        </div>

        <nav className="app__nav" aria-label="Primary navigation">
          <NavLink to="/chat" className={({ isActive }) => (isActive ? 'active' : '')}>
            <ChatIcon />
            <span>Chat</span>
          </NavLink>
          <NavLink to="/simulator" className={({ isActive }) => (isActive ? 'active' : '')}>
            <PondIcon />
            <span className="nav-label--wide">Farm Simulator</span>
            <span className="nav-label--compact">Farm</span>
          </NavLink>
        </nav>

        <div className="app__status">
          <span className="app__environment"><span aria-hidden="true" /> Development</span>
          <code title="AquaDoc API endpoint">{config.baseUrl}</code>
          {!config.devToken && (
            <span className="app__warning" title="Set VITE_AQUADOC_DEV_TOKEN in .env">
              No dev token
            </span>
          )}
          {!debugAvailable && <span className="app__muted">Debug panels disabled</span>}
        </div>
      </header>

      <main className="app__main">
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<Navigate to="/chat" replace />} />
            <Route path="/chat" element={<ChatPage />} />
            <Route path="/simulator" element={<FarmSimulatorPage />} />
            <Route path="*" element={<Navigate to="/chat" replace />} />
          </Routes>
        </ErrorBoundary>
      </main>

      <footer className="app__footer">
        <p><strong>Smart Aqua collects.</strong> AquaDoc understands. You decide.</p>
        <span>Decision support only — not a veterinarian, laboratory, or device controller.</span>
      </footer>
    </div>
  )
}

function BrandMark() {
  return (
    <span className="brand-mark" aria-hidden="true">
      <svg viewBox="0 0 48 48">
        <path d="M8 24c7-8 15-11 25-6l7-5-1 10 1 10-7-5c-10 5-18 2-25-4Z" />
        <path d="M13 24c6 3 12 3 19 0" />
        <circle cx="31.5" cy="20.5" r="1.6" />
        <circle cx="18" cy="20" r="1.4" />
        <path d="M18 20 24 24l7.5-3.5" />
      </svg>
    </span>
  )
}

function ChatIcon() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M20 11.5a7.5 7.5 0 0 1-8 7.5 9 9 0 0 1-3.8-.8L4 20l1.4-4A7.4 7.4 0 0 1 4 11.5a8 8 0 0 1 16 0Z" />
      <path d="M8 12h.01M12 12h.01M16 12h.01" />
    </svg>
  )
}

function PondIcon() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 7.5c3-2 6-2 9 0s6 2 9 0M3 12c3-2 6-2 9 0s6 2 9 0M3 16.5c3-2 6-2 9 0s6 2 9 0" />
    </svg>
  )
}
