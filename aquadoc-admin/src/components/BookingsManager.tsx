import React, { useState } from 'react'
import { Booking } from '../types'

interface Props {
  bookings: Booking[]
  onUpdateBooking: (id: string, update: Partial<Booking>) => Promise<void>
  loading: boolean
}

export const BookingsManager: React.FC<Props> = ({
  bookings,
  onUpdateBooking,
  loading,
}) => {
  const [filter, setFilter] = useState<string>('all')
  const [selectedBooking, setSelectedBooking] = useState<Booking | null>(null)
  const [newStatus, setNewStatus] = useState<Booking['status']>('pending')
  const [assignedVet, setAssignedVet] = useState<string>('')
  const [vetNotes, setVetNotes] = useState<string>('')
  const [saving, setSaving] = useState(false)

  const filteredBookings = bookings.filter((b) => {
    if (filter === 'all') return true
    return b.status === filter
  })

  const openManageModal = (b: Booking) => {
    setSelectedBooking(b)
    setNewStatus(b.status)
    setAssignedVet(b.assigned_vet || 'Dr. Chinedu Okafor (Field Pathologist)')
    setVetNotes(b.notes || '')
  }

  const handleSaveModal = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedBooking) return

    setSaving(true)
    await onUpdateBooking(selectedBooking.id, {
      status: newStatus,
      assigned_vet: assignedVet,
      notes: vetNotes,
    })
    setSaving(false)
    setSelectedBooking(null)
  }

  return (
    <div className="bookings-section">
      {/* Header & Filter Toolbar */}
      <div className="bookings-toolbar">
        <div>
          <h2 style={{ fontFamily: 'var(--font-heading)', fontSize: '1.25rem', margin: 0 }}>
            🚜 Veterinary Inspection & Consultation Queue
          </h2>
          <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
            Real-time management for emergency on-farm dispatches and live tele-consultations
          </span>
        </div>

        <div className="filters-group">
          {(['all', 'pending', 'confirmed', 'dispatched', 'completed'] as const).map((st) => (
            <button
              key={st}
              type="button"
              className={`filter-btn ${filter === st ? 'filter-btn--active' : ''}`}
              onClick={() => setFilter(st)}
            >
              {st.charAt(0).toUpperCase() + st.slice(1)} (
              {st === 'all' ? bookings.length : bookings.filter((b) => b.status === st).length})
            </button>
          ))}
        </div>
      </div>

      {/* Bookings Table */}
      <div className="table-card">
        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: 'var(--text-muted)' }}>
            Loading bookings queue…
          </div>
        ) : filteredBookings.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: 'var(--text-muted)' }}>
            No consultation bookings found under <strong>{filter}</strong> status.
          </div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Booking ID</th>
                <th>Farmer & Contact</th>
                <th>Farm Location</th>
                <th>Type & Species</th>
                <th>Symptoms / Case Notes</th>
                <th>Scheduled Time</th>
                <th>Assigned Vet</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredBookings.map((b) => (
                <tr key={b.id}>
                  <td>
                    <code>{b.id}</code>
                  </td>
                  <td>
                    <strong>{b.farmer_name}</strong>
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)' }}>
                      {b.farmer_phone}
                    </div>
                  </td>
                  <td style={{ maxWidth: '180px' }}>
                    <span style={{ fontSize: '0.84rem' }}>{b.farm_location}</span>
                  </td>
                  <td>
                    <span style={{ fontSize: '0.84rem', fontWeight: 600, display: 'block' }}>
                      {b.booking_type === 'physical' ? '🚜 On-Farm Physical' : '📹 Live Tele-Vet'}
                    </span>
                    <small style={{ color: 'var(--text-muted)' }}>{b.species}</small>
                  </td>
                  <td style={{ maxWidth: '240px' }}>
                    <div style={{ fontSize: '0.8rem', lineHeight: 1.3 }}>
                      {b.symptoms.length > 0 ? b.symptoms.join(', ') : b.notes || 'Routine check'}
                    </div>
                  </td>
                  <td>
                    <span style={{ fontSize: '0.84rem' }}>{b.preferred_date}</span>
                  </td>
                  <td>
                    <span style={{ fontSize: '0.82rem', color: b.assigned_vet ? 'var(--accent-cyan)' : 'var(--text-muted)' }}>
                      {b.assigned_vet || 'Unassigned'}
                    </span>
                  </td>
                  <td>
                    <span className={`status-badge status-badge--${b.status}`}>
                      {b.status}
                    </span>
                  </td>
                  <td>
                    <div className="action-buttons-cell">
                      <button
                        type="button"
                        className="btn-small btn-small--primary"
                        onClick={() => openManageModal(b)}
                        title="Manage status and assign veterinarian"
                      >
                        Manage
                      </button>
                      <a
                        href={`https://wa.me/${b.farmer_phone.replace(/\D/g, '')}?text=Hello%20${encodeURIComponent(b.farmer_name)},%20this%20is%20the%20Smart%20Aqua%20Veterinary%20Team%20regarding%20your%20consultation%20booking%20${b.id}`}
                        target="_blank"
                        rel="noreferrer"
                        className="btn-small btn-small--whatsapp"
                        title="Direct WhatsApp"
                      >
                        💬
                      </a>
                      <a
                        href={`tel:${b.farmer_phone}`}
                        className="btn-small"
                        title="Call Phone"
                      >
                        📞
                      </a>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Modal for Managing Booking */}
      {selectedBooking && (
        <div className="modal-overlay" onClick={() => setSelectedBooking(null)}>
          <div className="modal-dialog" onClick={(e) => e.stopPropagation()}>
            <h3 className="modal-title">
              Manage Booking: <code>{selectedBooking.id}</code>
            </h3>

            <div className="detail-row">
              <span className="detail-label">Farmer & Location</span>
              <strong>{selectedBooking.farmer_name} ({selectedBooking.farmer_phone})</strong>
              <p style={{ margin: '2px 0 0', color: 'var(--text-muted)' }}>{selectedBooking.farm_location}</p>
            </div>

            <div className="detail-row">
              <span className="detail-label">Service & Stock</span>
              <strong>{selectedBooking.booking_type === 'physical' ? 'On-Farm Inspection' : 'Live Video Call'} — {selectedBooking.species}</strong>
            </div>

            <form onSubmit={handleSaveModal}>
              <div className="form-field-group">
                <label>Booking Status</label>
                <select
                  value={newStatus}
                  onChange={(e) => setNewStatus(e.target.value as Booking['status'])}
                  className="admin-select"
                >
                  <option value="pending">Pending</option>
                  <option value="confirmed">Confirmed</option>
                  <option value="dispatched">Dispatched (Field Vet on Route)</option>
                  <option value="completed">Completed</option>
                  <option value="cancelled">Cancelled</option>
                </select>
              </div>

              <div className="form-field-group">
                <label>Assigned Aquatic Veterinarian</label>
                <select
                  value={assignedVet}
                  onChange={(e) => setAssignedVet(e.target.value)}
                  className="admin-select"
                >
                  <option value="Dr. Chinedu Okafor (Field Pathologist)">Dr. Chinedu Okafor (Field Pathologist - Lagos/Ogun)</option>
                  <option value="Dr. Amina Bello (Water Quality Specialist)">Dr. Amina Bello (Water Quality Specialist - Oyo/Ibadan)</option>
                  <option value="Dr. Emeka Nze (Aquatic Vet)">Dr. Emeka Nze (Aquatic Vet - Delta/Rivers)</option>
                  <option value="Dr. Folake Adeyemi (Nutrition & Parasitology)">Dr. Folake Adeyemi (Nutrition & Parasitology)</option>
                </select>
              </div>

              <div className="form-field-group">
                <label>Veterinary Clinical & Dispatch Notes</label>
                <textarea
                  rows={3}
                  value={vetNotes}
                  onChange={(e) => setVetNotes(e.target.value)}
                  className="admin-input"
                  style={{ resize: 'vertical' }}
                  placeholder="e.g., Dispatched for on-site water test & necropsy. Advised 3 ppt salt bath in interim."
                />
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
                <button
                  type="button"
                  className="btn-small"
                  onClick={() => setSelectedBooking(null)}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-small btn-small--primary"
                  disabled={saving}
                >
                  {saving ? 'Updating…' : 'Save & Update Booking'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
