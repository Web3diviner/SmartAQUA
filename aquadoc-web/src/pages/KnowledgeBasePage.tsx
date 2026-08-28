/**
 * Knowledge Base Explorer Page.
 *
 * 15_AQUADOC_FRONTEND.md section 4: Admin/Development RAG document repository.
 * Allows developers and experts to filter documents by status, species, and topic,
 * inspect metadata, vector chunk counts, and simulate document review workflows.
 */

import { useMemo, useState } from 'react'

export interface KnowledgeDocument {
  id: string
  title: string
  source_name: string
  author: string | null
  year: number
  species: string[]
  topics: string[]
  chunk_count: number
  status: 'approved' | 'pending' | 'deprecated' | 'rejected'
  uploaded_at: string
  summary: string
}

const MOCK_DOCUMENTS: KnowledgeDocument[] = [
  {
    id: 'doc-fao-01',
    title: 'FAO African Catfish Production & Water Quality Manual',
    source_name: 'FAO Agriculture Series',
    author: 'Food and Agriculture Organization',
    year: 2022,
    species: ['Clarias gariepinus'],
    topics: ['Water Quality', 'Feeding Protocols', 'Dissolved Oxygen'],
    chunk_count: 142,
    status: 'approved',
    uploaded_at: '2026-07-15',
    summary: 'Comprehensive guide covering optimal feeding conversion ratios (FCR), thermal tolerance, and emergency aeration protocols.',
  },
  {
    id: 'doc-tilapia-02',
    title: 'Nile Tilapia Broodstock & Hatchery Guidelines',
    source_name: 'WorldFish Center',
    author: 'Dr. M. Abubakar',
    year: 2023,
    species: ['Oreochromis niloticus'],
    topics: ['Breeding', 'Water Temperature', 'pH Management'],
    chunk_count: 98,
    status: 'approved',
    uploaded_at: '2026-08-01',
    summary: 'Standard operational guidelines for hatchery water clarity, salinity control, and fry feeding intervals.',
  },
  {
    id: 'doc-health-03',
    title: 'Bacterial and Parasitic Pathogens in Sub-Saharan Pond Farming',
    source_name: 'Journal of Aquaculture Health',
    author: 'Dr. E. O. Okonjo et al.',
    year: 2024,
    species: ['Clarias gariepinus', 'Oreochromis niloticus'],
    topics: ['Disease Diagnostics', 'Antibiotic Guidelines', 'Biosecurity'],
    chunk_count: 116,
    status: 'approved',
    uploaded_at: '2026-08-10',
    summary: 'Diagnostic matrix for Motile Aeromonas Septicemia (MAS), columnaris, and skin lesion triage.',
  },
  {
    id: 'doc-draft-04',
    title: 'Experimental Solar-Powered Aeration & Dissolved Oxygen Profiles',
    source_name: 'Smart Aqua Internal R&D',
    author: 'Smart Aqua Engineering Team',
    year: 2026,
    species: ['Clarias gariepinus'],
    topics: ['Aeration', 'IoT Sensors', 'Dissolved Oxygen'],
    chunk_count: 45,
    status: 'pending',
    uploaded_at: '2026-08-25',
    summary: 'Field evaluation of night aeration cycles and dissolved oxygen recovery rates in concrete tanks.',
  },
  {
    id: 'doc-legacy-05',
    title: 'Historical Feeding Chart (Outdated Standard 2018)',
    source_name: 'Legacy Farm Manual',
    author: 'Anon',
    year: 2018,
    species: ['Clarias gariepinus'],
    topics: ['Feeding Protocols'],
    chunk_count: 22,
    status: 'deprecated',
    uploaded_at: '2025-01-10',
    summary: 'Deprecating legacy static feed table due to lack of temperature-adjusted feeding coefficients.',
  },
]

export function KnowledgeBasePage() {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedStatus, setSelectedStatus] = useState<string>('all')
  const [selectedTopic, setSelectedTopic] = useState<string>('all')
  const [documents, setDocuments] = useState<KnowledgeDocument[]>(MOCK_DOCUMENTS)
  const [activeDoc, setActiveDoc] = useState<KnowledgeDocument | null>(null)

  const filteredDocs = useMemo(() => {
    return documents.filter((doc) => {
      const matchesSearch =
        doc.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
        doc.summary.toLowerCase().includes(searchTerm.toLowerCase()) ||
        doc.source_name.toLowerCase().includes(searchTerm.toLowerCase())

      const matchesStatus = selectedStatus === 'all' || doc.status === selectedStatus
      const matchesTopic = selectedTopic === 'all' || doc.topics.includes(selectedTopic)

      return matchesSearch && matchesStatus && matchesTopic
    })
  }, [documents, searchTerm, selectedStatus, selectedTopic])

  const handleUpdateStatus = (id: string, newStatus: KnowledgeDocument['status']) => {
    setDocuments((prev) =>
      prev.map((d) => (d.id === id ? { ...d, status: newStatus } : d)),
    )
    if (activeDoc?.id === id) {
      setActiveDoc((prev) => (prev ? { ...prev, status: newStatus } : null))
    }
  }

  return (
    <div className="knowledge-page">
      <header className="page-header">
        <div>
          <span className="page-eyebrow">RAG Knowledge Engine</span>
          <h2>Grounding Knowledge Base</h2>
        </div>
        <p>
          Curated aquaculture literature, FAO manuals, and validated research used by AquaDoc for RAG retrieval.
        </p>
      </header>

      <div className="knowledge-controls">
        <div className="search-bar">
          <svg viewBox="0 0 24 24" className="icon" aria-hidden="true">
            <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="2" fill="none" />
            <path d="M20 20L16 16" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
          </svg>
          <input
            type="text"
            aria-label="Search documents, topics, authors"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>

        <div className="filter-group">
          <select value={selectedStatus} onChange={(e) => setSelectedStatus(e.target.value)}>
            <option value="all">All Statuses</option>
            <option value="approved">Approved (Active RAG)</option>
            <option value="pending">Pending Review</option>
            <option value="deprecated">Deprecated</option>
            <option value="rejected">Rejected</option>
          </select>

          <select value={selectedTopic} onChange={(e) => setSelectedTopic(e.target.value)}>
            <option value="all">All Topics</option>
            <option value="Water Quality">Water Quality</option>
            <option value="Feeding Protocols">Feeding Protocols</option>
            <option value="Disease Diagnostics">Disease Diagnostics</option>
            <option value="Aeration">Aeration</option>
          </select>
        </div>
      </div>

      <div className="knowledge-grid">
        <div className="doc-list">
          {filteredDocs.length === 0 ? (
            <div className="empty-state">
              <p>No documents found matching the filter criteria.</p>
            </div>
          ) : (
            filteredDocs.map((doc) => (
              <article
                key={doc.id}
                className={`doc-card ${activeDoc?.id === doc.id ? 'doc-card--active' : ''}`}
                onClick={() => setActiveDoc(doc)}
              >
                <div className="doc-card__header">
                  <span className={`status-pill status-pill--${doc.status}`}>
                    {doc.status}
                  </span>
                  <span className="doc-card__year">{doc.year}</span>
                </div>

                <h3 className="doc-card__title">{doc.title}</h3>
                <p className="doc-card__source">{doc.source_name}</p>

                <div className="doc-card__tags">
                  {doc.topics.map((topic) => (
                    <span key={topic} className="tag">
                      {topic}
                    </span>
                  ))}
                </div>

                <div className="doc-card__footer">
                  <span>{doc.chunk_count} vector chunks</span>
                  <span>{doc.species.length} species</span>
                </div>
              </article>
            ))
          )}
        </div>

        <aside className="doc-detail-panel">
          {activeDoc ? (
            <div className="detail-content">
              <div className="detail-header">
                <span className={`status-pill status-pill--${activeDoc.status}`}>
                  {activeDoc.status.toUpperCase()}
                </span>
                <span className="detail-id">{activeDoc.id}</span>
              </div>

              <h2>{activeDoc.title}</h2>
              <p className="detail-author">
                By <strong>{activeDoc.author || 'Unknown'}</strong> &bull; {activeDoc.source_name} ({activeDoc.year})
              </p>

              <div className="detail-section">
                <h4>Summary</h4>
                <p>{activeDoc.summary}</p>
              </div>

              <div className="detail-section">
                <h4>Target Species</h4>
                <div className="tag-cloud">
                  {activeDoc.species.map((s) => (
                    <span key={s} className="tag tag--accent">
                      {s}
                    </span>
                  ))}
                </div>
              </div>

              <div className="detail-section">
                <h4>RAG Vector Metrics</h4>
                <dl className="metrics-list">
                  <div>
                    <dt>Chunk Count</dt>
                    <dd>{activeDoc.chunk_count} embeddings</dd>
                  </div>
                  <div>
                    <dt>Embedding Model</dt>
                    <dd>voyage-3-large</dd>
                  </div>
                  <div>
                    <dt>Upload Date</dt>
                    <dd>{activeDoc.uploaded_at}</dd>
                  </div>
                </dl>
              </div>

              <div className="detail-actions">
                <h4>Admin Actions</h4>
                <div className="button-row">
                  {activeDoc.status !== 'approved' && (
                    <button
                      type="button"
                      className="button button--primary"
                      onClick={() => handleUpdateStatus(activeDoc.id, 'approved')}
                    >
                      Approve for RAG
                    </button>
                  )}
                  {activeDoc.status !== 'deprecated' && (
                    <button
                      type="button"
                      className="button"
                      onClick={() => handleUpdateStatus(activeDoc.id, 'deprecated')}
                    >
                      Deprecate
                    </button>
                  )}
                  {activeDoc.status !== 'rejected' && (
                    <button
                      type="button"
                      className="button button--danger"
                      onClick={() => handleUpdateStatus(activeDoc.id, 'rejected')}
                    >
                      Reject
                    </button>
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="detail-empty-state">
              <svg viewBox="0 0 24 24" className="empty-state-icon" aria-hidden="true">
                <path
                  d="M14 2H6C4.9 2 4 2.9 4 4V20C4 21.1 4.9 22 6 22H18C19.1 22 20 21.1 20 20V8L14 2ZM18 20H6V4H13V9H18V20Z"
                  fill="currentColor"
                />
              </svg>
              <h3>Select a document</h3>
              <p>Click any document on the left to inspect its vector metadata, species tags, and approval state.</p>
            </div>
          )}
        </aside>
      </div>
    </div>
  )
}
