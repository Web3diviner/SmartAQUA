import React, { useState } from 'react'
import { AdminAnalyticsResponse } from '../types'

interface Props {
  data: AdminAnalyticsResponse | null
  loading: boolean
}

export const AnalyticsOverview: React.FC<Props> = ({ data, loading }) => {
  const [hoveredDay, setHoveredDay] = useState<number | null>(null)

  if (loading || !data) {
    return (
      <div style={{ padding: '60px', textAlign: 'center', color: 'var(--text-muted)' }}>
        Loading platform telemetry & user growth analytics…
      </div>
    )
  }

  const { kpis, daily_users_trend, regional_distribution, top_diagnosed_conditions } = data

  // Chart dimensions & polyline calculations
  const chartHeight = 220
  const chartWidth = 720
  const maxDau = Math.max(...daily_users_trend.map((d) => d.active_users)) * 1.15
  const maxNew = Math.max(...daily_users_trend.map((d) => d.new_onboarded)) * 1.4

  const dauPoints = daily_users_trend
    .map((d, i) => {
      const x = (i / (daily_users_trend.length - 1)) * (chartWidth - 60) + 40
      const y = chartHeight - 30 - (d.active_users / maxDau) * (chartHeight - 60)
      return `${x},${y}`
    })
    .join(' ')

  const newPoints = daily_users_trend
    .map((d, i) => {
      const x = (i / (daily_users_trend.length - 1)) * (chartWidth - 60) + 40
      const y = chartHeight - 30 - (d.new_onboarded / maxNew) * (chartHeight - 60)
      return `${x},${y}`
    })
    .join(' ')

  const activeTrend = hoveredDay !== null
    ? daily_users_trend[hoveredDay]
    : daily_users_trend[daily_users_trend.length - 1]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '28px' }}>
      {/* KPI Summary Cards */}
      <div className="kpi-grid">
        <div className="kpi-card">
          <div className="kpi-card__top">
            <span className="kpi-card__label">Users Onboarded</span>
            <span className="kpi-card__icon">👥</span>
          </div>
          <div className="kpi-card__value">{kpis.total_users_onboarded.toLocaleString()}</div>
          <div className="kpi-card__growth kpi-card__growth--up">
            ▲ +{kpis.onboarded_growth_mom_pct}% MoM growth
          </div>
        </div>

        <div className="kpi-card">
          <div className="kpi-card__top">
            <span className="kpi-card__label">Daily Active Users (DAU)</span>
            <span className="kpi-card__icon">⚡</span>
          </div>
          <div className="kpi-card__value" style={{ color: 'var(--accent-cyan)' }}>
            {kpis.daily_active_users.toLocaleString()}
          </div>
          <div className="kpi-card__growth kpi-card__growth--up">
            ▲ +{kpis.dau_growth_wow_pct}% vs last week
          </div>
        </div>

        <div className="kpi-card">
          <div className="kpi-card__top">
            <span className="kpi-card__label">Ponds Monitored</span>
            <span className="kpi-card__icon">🌊</span>
          </div>
          <div className="kpi-card__value">{kpis.total_ponds_monitored.toLocaleString()}</div>
          <div className="kpi-card__growth kpi-card__growth--neutral">
            Across {regional_distribution.length} Agricultural Zones
          </div>
        </div>

        <div className="kpi-card">
          <div className="kpi-card__top">
            <span className="kpi-card__label">Triage & Bookings</span>
            <span className="kpi-card__icon">🚜</span>
          </div>
          <div className="kpi-card__value">{kpis.total_bookings_count}</div>
          <div className="kpi-card__growth" style={{ color: kpis.pending_bookings_count > 0 ? '#f59e0b' : '#10b981' }}>
            {kpis.pending_bookings_count} pending dispatch
          </div>
        </div>
      </div>

      {/* Analytics Main Layout */}
      <div className="analytics-layout">
        {/* Left Column: 14-Day DAU & Onboarding Curve */}
        <div className="chart-card">
          <div className="chart-card__header">
            <div>
              <h3 className="chart-card__title">📈 14-Day Daily Active Users & Onboarding Velocity</h3>
              <span className="chart-card__sub">Continuous engagement and growth tracking</span>
            </div>
            <div className="chart-legend">
              <span className="legend-item">
                <span className="legend-dot legend-dot--dau" /> Daily Active Users (DAU)
              </span>
              <span className="legend-item">
                <span className="legend-dot legend-dot--new" /> Daily New Onboarded
              </span>
            </div>
          </div>

          <div className="dau-chart-container">
            <svg
              viewBox={`0 0 ${chartWidth} ${chartHeight}`}
              className="dau-svg"
              onMouseLeave={() => setHoveredDay(null)}
            >
              {/* Grid Lines */}
              <line x1="40" y1={chartHeight - 30} x2={chartWidth - 20} y2={chartHeight - 30} stroke="rgba(255,255,255,0.08)" />
              <line x1="40" y1={chartHeight / 2} x2={chartWidth - 20} y2={chartHeight / 2} stroke="rgba(255,255,255,0.04)" />
              <line x1="40" y1="30" x2={chartWidth - 20} y2="30" stroke="rgba(255,255,255,0.04)" />

              {/* Polylines */}
              <polyline
                fill="none"
                stroke="#a855f7"
                strokeWidth="2.5"
                strokeDasharray="3 3"
                points={newPoints}
              />

              <polyline
                fill="none"
                stroke="#06b6d4"
                strokeWidth="3.5"
                points={dauPoints}
              />

              {/* Hover nodes */}
              {daily_users_trend.map((d, i) => {
                const x = (i / (daily_users_trend.length - 1)) * (chartWidth - 60) + 40
                const y = chartHeight - 30 - (d.active_users / maxDau) * (chartHeight - 60)
                return (
                  <circle
                    key={d.date}
                    cx={x}
                    cy={y}
                    r={hoveredDay === i ? 6 : 3.5}
                    fill={hoveredDay === i ? '#ffffff' : '#06b6d4'}
                    stroke="#06b6d4"
                    strokeWidth="2"
                    style={{ cursor: 'pointer' }}
                    onMouseEnter={() => setHoveredDay(i)}
                  />
                )
              })}
            </svg>

            {activeTrend && (
              <div className="chart-inspector">
                <div className="insp-item">
                  <span className="insp-label">Date</span>
                  <strong className="insp-val">{activeTrend.date}</strong>
                </div>
                <div className="insp-item">
                  <span className="insp-label">Active Farmers</span>
                  <strong className="insp-val" style={{ color: 'var(--accent-cyan)' }}>
                    {activeTrend.active_users}
                  </strong>
                </div>
                <div className="insp-item">
                  <span className="insp-label">New Onboarded</span>
                  <strong className="insp-val" style={{ color: 'var(--accent-purple)' }}>
                    +{activeTrend.new_onboarded}
                  </strong>
                </div>
                <div className="insp-item">
                  <span className="insp-label">System Load</span>
                  <strong className="insp-val" style={{ color: 'var(--accent-emerald)' }}>
                    Optimal
                  </strong>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Regional Hubs Breakdown */}
        <div className="breakdown-card">
          <div>
            <h3 className="chart-card__title">📍 Regional Distribution</h3>
            <span className="chart-card__sub">Registered farm density across aquaculture hubs</span>
          </div>

          <div className="breakdown-list">
            {regional_distribution.map((reg) => (
              <div key={reg.region} className="breakdown-item">
                <div className="breakdown-header">
                  <span>{reg.region}</span>
                  <strong>{reg.count} ({reg.percentage}%)</strong>
                </div>
                <div className="breakdown-bar-bg">
                  <div
                    className="breakdown-bar-fill"
                    style={{ width: `${reg.percentage}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Bottom Grid: Top Clinical Conditions Diagnosed */}
      <div className="breakdown-card">
        <div>
          <h3 className="chart-card__title">🩺 Top Diagnosed Conditions & Pathologies</h3>
          <span className="chart-card__sub">Aggregated diagnostic frequency from farm triage reports</span>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '16px' }}>
          {top_diagnosed_conditions.map((item) => (
            <div
              key={item.condition}
              style={{
                padding: '16px',
                background: 'var(--bg-surface-subtle)',
                borderRadius: 'var(--radius-md)',
                border: '1px solid var(--border-app)',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <div>
                <strong style={{ fontSize: '0.9rem', display: 'block' }}>{item.condition}</strong>
                <small style={{ color: 'var(--text-muted)' }}>{item.cases} verified farm cases</small>
              </div>
              <span
                style={{
                  fontSize: '0.72rem',
                  padding: '3px 8px',
                  borderRadius: '9999px',
                  fontWeight: 700,
                  textTransform: 'uppercase',
                  background: item.severity === 'critical' ? 'rgba(239, 68, 68, 0.15)' : 'rgba(245, 158, 11, 0.15)',
                  color: item.severity === 'critical' ? '#ef4444' : '#f59e0b',
                }}
              >
                {item.severity}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
