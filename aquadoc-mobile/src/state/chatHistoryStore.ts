/**
 * Persistent Chat History Store.
 *
 * Manages multiple consultation threads, saves turns to localStorage,
 * and handles chronological organization (Today, Yesterday, Previous 7 Days, Older).
 */

import type { ChatResponse } from '@/schemas/aquadoc'

export interface ChatTurn {
  id: string
  question: string
  response: ChatResponse | null
  error: string | null
}

export interface ChatSession {
  id: string
  title: string
  createdAt: string
  updatedAt: string
  turns: ChatTurn[]
}

const STORAGE_KEY = 'smartaqua_chat_sessions_v1'

export function loadSavedSessions(): ChatSession[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as ChatSession[]
    return Array.isArray(parsed) ? parsed : []
  } catch (err) {
    console.warn('Failed to parse chat sessions from storage:', err)
    return []
  }
}

export function saveSessionsToStorage(sessions: ChatSession[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions))
  } catch (err) {
    console.warn('Failed to save chat sessions to storage:', err)
  }
}

export function createNewSession(initialQuestion?: string): ChatSession {
  const id = `conv-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`
  const now = new Date().toISOString()
  let title = 'New AquaDoc Consultation'
  if (initialQuestion && initialQuestion.trim()) {
    const clean = initialQuestion.trim().replace(/\n+/g, ' ')
    title = clean.length > 50 ? `${clean.substring(0, 47)}...` : clean
  }

  return {
    id,
    title,
    createdAt: now,
    updatedAt: now,
    turns: [],
  }
}

export function formatSessionRelativeTime(isoString: string): string {
  if (!isoString) return ''
  try {
    const date = new Date(isoString)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / (1000 * 60))
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays === 1) return 'Yesterday'
    if (diffDays < 7) return `${diffDays}d ago`
    return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  } catch {
    return ''
  }
}

export function groupSessionsByDate(sessions: ChatSession[]): { label: string; sessions: ChatSession[] }[] {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const yesterday = today - 86400000
  const sevenDaysAgo = today - 86400000 * 7

  const todayList: ChatSession[] = []
  const yesterdayList: ChatSession[] = []
  const sevenDaysList: ChatSession[] = []
  const olderList: ChatSession[] = []

  const sorted = [...sessions].sort(
    (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
  )

  for (const session of sorted) {
    const sessionTime = new Date(session.updatedAt || session.createdAt).getTime()
    if (sessionTime >= today) {
      todayList.push(session)
    } else if (sessionTime >= yesterday) {
      yesterdayList.push(session)
    } else if (sessionTime >= sevenDaysAgo) {
      sevenDaysList.push(session)
    } else {
      olderList.push(session)
    }
  }

  const result: { label: string; sessions: ChatSession[] }[] = []
  if (todayList.length > 0) result.push({ label: 'Today', sessions: todayList })
  if (yesterdayList.length > 0) result.push({ label: 'Yesterday', sessions: yesterdayList })
  if (sevenDaysList.length > 0) result.push({ label: 'Previous 7 Days', sessions: sevenDaysList })
  if (olderList.length > 0) result.push({ label: 'Older', sessions: olderList })

  return result
}

/**
 * Cloud Synchronization API helpers for multi-device history persistence.
 */

export async function fetchUserSessionsFromCloud(baseUrl: string, userId: string): Promise<ChatSession[]> {
  try {
    const res = await fetch(`${baseUrl}/dev/v1/users/${encodeURIComponent(userId)}/chat-sessions`)
    if (!res.ok) return []
    const data = await res.json()
    return Array.isArray(data.sessions) ? data.sessions : []
  } catch (err) {
    console.warn('Failed to fetch cloud sessions:', err)
    return []
  }
}

export async function syncSessionsWithCloud(
  baseUrl: string,
  userId: string,
  localSessions: ChatSession[],
): Promise<ChatSession[]> {
  try {
    const res = await fetch(`${baseUrl}/dev/v1/users/${encodeURIComponent(userId)}/chat-sessions/sync`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sessions: localSessions }),
    })
    if (!res.ok) return localSessions
    const data = await res.json()
    if (Array.isArray(data.sessions)) {
      saveSessionsToStorage(data.sessions)
      return data.sessions
    }
    return localSessions
  } catch (err) {
    console.warn('Failed to sync sessions with cloud:', err)
    return localSessions
  }
}

export async function saveSessionToCloud(baseUrl: string, userId: string, session: ChatSession): Promise<void> {
  try {
    await fetch(`${baseUrl}/dev/v1/users/${encodeURIComponent(userId)}/chat-sessions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(session),
    })
  } catch (err) {
    console.warn('Failed to save session to cloud:', err)
  }
}

export async function deleteSessionFromCloud(baseUrl: string, userId: string, sessionId: string): Promise<void> {
  try {
    await fetch(`${baseUrl}/dev/v1/users/${encodeURIComponent(userId)}/chat-sessions/${encodeURIComponent(sessionId)}`, {
      method: 'DELETE',
    })
  } catch (err) {
    console.warn('Failed to delete session from cloud:', err)
  }
}

export async function clearAllSessionsFromCloud(baseUrl: string, userId: string): Promise<void> {
  try {
    await fetch(`${baseUrl}/dev/v1/users/${encodeURIComponent(userId)}/chat-sessions`, {
      method: 'DELETE',
    })
  } catch (err) {
    console.warn('Failed to clear cloud sessions:', err)
  }
}
