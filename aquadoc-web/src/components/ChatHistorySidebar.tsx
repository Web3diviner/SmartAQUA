import React, { useState } from 'react'
import { useAppState } from '@/app/providers'
import {
  type ChatSession,
  formatSessionRelativeTime,
  groupSessionsByDate,
} from '@/state/chatHistoryStore'

interface ChatHistorySidebarProps {
  isOpen: boolean
  onClose: () => void
  sessions: ChatSession[]
  activeSessionId: string | null
  onSelectSession: (session: ChatSession) => void
  onNewSession: () => void
  onDeleteSession: (sessionId: string, e: React.MouseEvent) => void
  onSync?: () => void
  isSyncing?: boolean
}

export const ChatHistorySidebar: React.FC<ChatHistorySidebarProps> = ({
  isOpen,
  onClose,
  sessions,
  activeSessionId,
  onSelectSession,
  onNewSession,
  onDeleteSession,
  onSync,
  isSyncing = false,
}) => {
  const { user } = useAppState()
  const [searchQuery, setSearchQuery] = useState('')

  const filteredSessions = sessions.filter((s) =>
    s.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    s.turns.some((t) => t.question.toLowerCase().includes(searchQuery.toLowerCase()))
  )

  const dateGroups = groupSessionsByDate(filteredSessions)

  return (
    <>
      {/* Mobile Backdrop */}
      {isOpen && (
        <div
          className="chat-history-backdrop"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      <aside className={`chat-history-sidebar ${isOpen ? 'open' : ''}`}>
        {/* Header */}
        <div className="chat-history-header">
          <div className="chat-history-title-wrap">
            <span className="chat-history-icon">💬</span>
            <div>
              <h3>Consultation History</h3>
              {user && (
                <span
                  style={{
                    fontSize: '0.68rem',
                    color: '#10b981',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontWeight: 600,
                  }}
                >
                  ☁️ Synced to {user.email ? user.email.split('@')[0] : 'cloud'}
                </span>
              )}
            </div>
          </div>
          <button
            type="button"
            className="chat-history-close-btn"
            onClick={onClose}
            title="Close sidebar"
            aria-label="Close chat history"
          >
            ✕
          </button>
        </div>

        {/* New Consultation & Sync Actions */}
        <div className="chat-history-actions">
          <button
            type="button"
            className="new-chat-btn"
            onClick={() => {
              onNewSession()
              if (window.innerWidth < 768) onClose()
            }}
          >
            <span className="btn-icon">✨</span>
            <span>New Consultation</span>
            <span className="btn-plus">+</span>
          </button>

          {onSync && user && (
            <button
              type="button"
              className="sync-cloud-btn"
              onClick={onSync}
              disabled={isSyncing}
              title="Force sync chat history across all your devices"
            >
              <span className={`btn-icon ${isSyncing ? 'spinning' : ''}`}>🔄</span>
              <span>{isSyncing ? 'Syncing Cloud...' : '🔄 Sync Device History'}</span>
            </button>
          )}
        </div>

        {/* Search Input */}
        {sessions.length > 3 && (
          <div className="chat-history-search">
            <span className="search-icon">🔍</span>
            <input
              type="text"
              placeholder="Search previous chats..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="chat-search-input"
            />
            {searchQuery && (
              <button
                type="button"
                className="clear-search-btn"
                onClick={() => setSearchQuery('')}
              >
                ✕
              </button>
            )}
          </div>
        )}

        {/* Sessions List */}
        <div className="chat-history-list">
          {sessions.length === 0 ? (
            <div className="chat-history-empty">
              <span className="empty-icon">📂</span>
              <p className="empty-title">No consultations yet</p>
              <p className="empty-subtitle">
                Ask AquaDoc any fish health or water question to start building your history.
              </p>
            </div>
          ) : filteredSessions.length === 0 ? (
            <div className="chat-history-empty">
              <p className="empty-title">No matching chats</p>
              <p className="empty-subtitle">Try searching for a different keyword.</p>
            </div>
          ) : (
            dateGroups.map((group) => (
              <div key={group.label} className="chat-history-group">
                <div className="group-heading">{group.label}</div>
                <div className="group-items">
                  {group.sessions.map((session) => {
                    const isActive = session.id === activeSessionId
                    const turnCount = session.turns.length

                    return (
                      <div
                        key={session.id}
                        className={`history-session-card ${isActive ? 'active' : ''}`}
                        onClick={() => {
                          onSelectSession(session)
                          if (window.innerWidth < 768) onClose()
                        }}
                        role="button"
                        tabIndex={0}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            onSelectSession(session)
                            if (window.innerWidth < 768) onClose()
                          }
                        }}
                      >
                        <div className="session-card-main">
                          <div className="session-title-row">
                            <span className="session-status-dot" />
                            <h4 className="session-title" title={session.title}>
                              {session.title}
                            </h4>
                          </div>
                          <div className="session-meta-row">
                            <span className="session-time">
                              {formatSessionRelativeTime(session.updatedAt || session.createdAt)}
                            </span>
                            {turnCount > 0 && (
                              <span className="session-turns-badge">
                                {turnCount} {turnCount === 1 ? 'turn' : 'turns'}
                              </span>
                            )}
                          </div>
                        </div>

                        <button
                          type="button"
                          className="session-delete-btn"
                          title="Delete consultation"
                          onClick={(e) => onDeleteSession(session.id, e)}
                          aria-label={`Delete ${session.title}`}
                        >
                          🗑️
                        </button>
                      </div>
                    )
                  })}
                </div>
              </div>
            ))
          )}
        </div>

        {/* Footer Info */}
        <div className="chat-history-footer">
          <span className="security-icon">🔒</span>
          <span>Conversations saved securely with continuous context memory.</span>
        </div>
      </aside>
    </>
  )
}
