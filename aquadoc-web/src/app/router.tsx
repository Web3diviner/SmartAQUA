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
import { AuthModal } from '@/components/AuthModal'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { ModelSelector } from '@/components/ModelSelector'
import { AuthPage } from '@/pages/AuthPage'
import { ChatPage } from '@/pages/ChatPage'
import { DiseaseTestPage } from '@/pages/DiseaseTestPage'
import { FarmSimulatorPage } from '@/pages/FarmSimulatorPage'

export function App() {
  const { theme, toggleTheme, user, isAuthenticated, openAuthModal, logout } = useAppState()

  return (
    <div className="app">
      <AuthModal />
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
          <NavLink to="/disease" className={({ isActive }) => (isActive ? 'active' : '')}>
            <HealthIcon />
            <span className="nav-label--wide">Disease Triage</span>
            <span className="nav-label--compact">Health</span>
          </NavLink>
        </nav>

        <div className="app__status">
          <ModelSelector />

          {/* User Auth Profile / Login Button */}
          {isAuthenticated && user ? (
            <div className="user-profile-badge">
              <img
                src={user.avatarUrl || `https://api.dicebear.com/7.x/bottts/svg?seed=${user.email}`}
                alt={user.name}
                className="user-avatar"
              />
              <div className="user-info-text">
                <strong className="user-name">{user.name}</strong>
                <small className="user-farm">{user.farmName || user.primarySpecies}</small>
              </div>
              <button
                type="button"
                className="btn-logout"
                onClick={logout}
                title="Log out of farm account"
              >
                Log Out
              </button>
            </div>
          ) : (
            <button
              type="button"
              className="button button--primary auth-login-btn"
              onClick={openAuthModal}
            >
              <span>👤 Sign In / Register</span>
            </button>
          )}

          <button
            type="button"
            className="theme-toggle-btn"
            onClick={toggleTheme}
            title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
          >
            {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
            <span>{theme === 'dark' ? 'Light' : 'Dark'}</span>
          </button>
        </div>
      </header>

      <main className="app__main">
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<Navigate to="/chat" replace />} />
            <Route path="/chat" element={<ChatPage />} />
            <Route path="/simulator" element={<FarmSimulatorPage />} />
            <Route path="/disease" element={<DiseaseTestPage />} />
            <Route path="/auth" element={<AuthPage />} />
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

function HealthIcon() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z" stroke="currentColor" strokeWidth="1.8" fill="none" />
    </svg>
  )
}


function SunIcon() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="5" stroke="currentColor" strokeWidth="1.8" fill="none" />
      <path d="M12 1v2m0 18v2M4.22 4.22l1.42 1.42m12.72 12.72l1.42 1.42M1 12h2m18 0h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" stroke="currentColor" strokeWidth="1.8" fill="none" />
    </svg>
  )
}

