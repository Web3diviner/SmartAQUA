import React, { useState, useEffect } from 'react'
import { Header } from './components/Header'
import { AnalyticsOverview } from './components/AnalyticsOverview'
import { BookingsManager } from './components/BookingsManager'
import { EvaluationHub } from './components/EvaluationHub'
import { AdminAnalyticsResponse, Booking, TraceMetric } from './types'

const getDefaultBaseUrl = () => {
  if (
    typeof window !== 'undefined' &&
    (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')
  ) {
    return 'http://127.0.0.1:8001'
  }
  return 'https://smartaqua-1.onrender.com'
}

const rawBaseUrl =
  import.meta.env.VITE_AQUADOC_BASE_URL ||
  import.meta.env.VITE_AQUADOC_API_URL ||
  getDefaultBaseUrl()

const isProductionBrowser =
  typeof window !== 'undefined' &&
  window.location.hostname !== 'localhost' &&
  window.location.hostname !== '127.0.0.1'

const BASE_URL = (
  isProductionBrowser && (rawBaseUrl.includes('127.0.0.1') || rawBaseUrl.includes('localhost'))
    ? getDefaultBaseUrl()
    : rawBaseUrl
).replace(/\/$/, '')

const DEV_TOKEN = import.meta.env.VITE_AQUADOC_DEV_TOKEN || 'aqua-dev-token-2026'

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'analytics' | 'bookings' | 'evaluation'>('analytics')
  const [analytics, setAnalytics] = useState<AdminAnalyticsResponse | null>(null)
  const [bookings, setBookings] = useState<Booking[]>([])
  const [traces, setTraces] = useState<TraceMetric[]>([])
  const [loading, setLoading] = useState<boolean>(true)

  const fetchAnalytics = async () => {
    try {
      const res = await fetch(`${BASE_URL}/dev/v1/admin/analytics`, {
        headers: { Authorization: `Bearer ${DEV_TOKEN}` },
      })
      if (res.ok) {
        const data = await res.json()
        setAnalytics(data)
      }
    } catch (err) {
      console.error('Failed to fetch admin analytics:', err)
    }
  }

  const fetchBookings = async () => {
    try {
      const res = await fetch(`${BASE_URL}/dev/v1/admin/bookings`, {
        headers: { Authorization: `Bearer ${DEV_TOKEN}` },
      })
      if (res.ok) {
        const data = await res.json()
        setBookings(data.bookings || [])
      }
    } catch (err) {
      console.error('Failed to fetch bookings:', err)
    }
  }

  const fetchTraces = async () => {
    try {
      const res = await fetch(`${BASE_URL}/dev/v1/admin/traces`, {
        headers: { Authorization: `Bearer ${DEV_TOKEN}` },
      })
      if (res.ok) {
        const data = await res.json()
        setTraces(data.traces || [])
      }
    } catch (err) {
      console.error('Failed to fetch evaluation traces:', err)
    }
  }

  const loadAll = async () => {
    setLoading(true)
    await Promise.all([fetchAnalytics(), fetchBookings(), fetchTraces()])
    setLoading(false)
  }

  useEffect(() => {
    loadAll()
    const interval = setInterval(loadAll, 15000) // Poll every 15s for live updates
    return () => clearInterval(interval)
  }, [])

  const handleUpdateBooking = async (id: string, update: Partial<Booking>) => {
    try {
      const res = await fetch(`${BASE_URL}/dev/v1/admin/bookings/${id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${DEV_TOKEN}`,
        },
        body: JSON.stringify(update),
      })
      if (res.ok) {
        await Promise.all([fetchBookings(), fetchAnalytics()])
      }
    } catch (err) {
      console.error('Failed to update booking:', err)
    }
  }

  const pendingCount = bookings.filter((b) => b.status === 'pending').length

  return (
    <div className="admin-app">
      <Header
        activeTab={activeTab}
        onTabChange={setActiveTab}
        pendingBookingsCount={pendingCount}
      />

      <main className="admin-main">
        {activeTab === 'analytics' && (
          <AnalyticsOverview data={analytics} loading={loading} />
        )}

        {activeTab === 'bookings' && (
          <BookingsManager
            bookings={bookings}
            onUpdateBooking={handleUpdateBooking}
            loading={loading}
          />
        )}

        {activeTab === 'evaluation' && (
          <EvaluationHub benchmarks={analytics?.system_benchmarks} traces={traces} />
        )}
      </main>

      <footer className="admin-footer">
        <div>
          <strong>AquaDoc AI Enterprise Admin Portal</strong> &bull; Smart Aqua Ecosystem
        </div>
        <div>
          Continuous Telemetry &bull; Port 5174 &bull; Backend: <code>{BASE_URL}</code>
        </div>
      </footer>
    </div>
  )
}
