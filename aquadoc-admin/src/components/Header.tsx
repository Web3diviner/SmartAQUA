import React from 'react'

interface Props {
  activeTab: 'analytics' | 'bookings' | 'evaluation'
  onTabChange: (tab: 'analytics' | 'bookings' | 'evaluation') => void
  pendingBookingsCount: number
}

export const Header: React.FC<Props> = ({
  activeTab,
  onTabChange,
  pendingBookingsCount,
}) => {
  return (
    <header className="admin-header">
      <div className="admin-header__brand">
        <div className="brand-icon">🛡️</div>
        <div className="brand-text">
          <h1>
            AquaDoc Operations
            <span className="admin-badge">Admin Portal</span>
          </h1>
          <span className="brand-sub">Smart Aqua Ecosystem Command Center</span>
        </div>
      </div>

      <nav className="admin-nav" aria-label="Admin navigation">
        <button
          type="button"
          className={`nav-tab ${activeTab === 'analytics' ? 'nav-tab--active' : ''}`}
          onClick={() => onTabChange('analytics')}
        >
          📈 User Growth & DAU
        </button>

        <button
          type="button"
          className={`nav-tab ${activeTab === 'bookings' ? 'nav-tab--active' : ''}`}
          onClick={() => onTabChange('bookings')}
        >
          🚜 Vet Inspection Bookings
          {pendingBookingsCount > 0 && (
            <span className="nav-counter">{pendingBookingsCount}</span>
          )}
        </button>

        <button
          type="button"
          className={`nav-tab ${activeTab === 'evaluation' ? 'nav-tab--active' : ''}`}
          onClick={() => onTabChange('evaluation')}
        >
          🧠 RAG Evaluation & Traces
        </button>
      </nav>

      <div className="admin-header__actions">
        <div className="live-pill">
          <span className="pulse-dot" />
          <span>Backend Live (:8001)</span>
        </div>

        <a
          href="http://localhost:5173"
          target="_blank"
          rel="noreferrer"
          className="farmer-app-link"
          title="Switch to Farmer Facing Web App"
        >
          🐟 Farmer App ↗
        </a>
      </div>
    </header>
  )
}
