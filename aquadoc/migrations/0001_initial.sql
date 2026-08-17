-- AquaDoc initial schema.
-- Knowledge + AquaDoc conversation domains from 03_DATA_MODEL.md.
--
-- Rollback: see 0001_initial_down.sql
--
-- NOTE: knowledge_chunks.embedding is declared vector(1024). This must match
-- EMBEDDING_DIMENSIONS. Changing the embedding model to a different dimension
-- requires a new migration plus a full re-ingest (embeddings are not portable
-- across models).

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- Knowledge domain
-- ---------------------------------------------------------------------------

CREATE TYPE knowledge_review_status AS ENUM ('pending', 'approved', 'deprecated', 'rejected');

-- A: official / expert-reviewed guideline
-- B: peer-reviewed research
-- C: established textbook / manual
-- D: verified Smart Aqua expert case
-- E: farmer / user report
CREATE TYPE knowledge_evidence_level AS ENUM ('A', 'B', 'C', 'D', 'E');

CREATE TABLE knowledge_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL,
    source          TEXT        NOT NULL,
    author          TEXT,
    year            INTEGER,
    document_type   TEXT        NOT NULL,
    species         TEXT[]      NOT NULL DEFAULT '{}',
    life_stage      TEXT[]      NOT NULL DEFAULT '{}',
    topic           TEXT[]      NOT NULL DEFAULT '{}',
    disease         TEXT[]      NOT NULL DEFAULT '{}',
    region          TEXT[]      NOT NULL DEFAULT '{}',
    evidence_level  knowledge_evidence_level NOT NULL,
    review_status   knowledge_review_status  NOT NULL DEFAULT 'pending',
    owner           TEXT,
    file_url        TEXT,
    checksum        TEXT        NOT NULL,
    version         INTEGER     NOT NULL DEFAULT 1,
    chunk_count     INTEGER     NOT NULL DEFAULT 0,
    ingest_status   TEXT        NOT NULL DEFAULT 'pending',
    ingest_error    TEXT,
    ingested_at     TIMESTAMPTZ,
    reviewed_at     TIMESTAMPTZ,
    reviewed_by     TEXT,
    review_note     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT knowledge_documents_checksum_version_key UNIQUE (checksum, version)
);

CREATE INDEX knowledge_documents_review_status_idx ON knowledge_documents (review_status);
CREATE INDEX knowledge_documents_species_idx        ON knowledge_documents USING GIN (species);
CREATE INDEX knowledge_documents_topic_idx          ON knowledge_documents USING GIN (topic);
CREATE INDEX knowledge_documents_disease_idx        ON knowledge_documents USING GIN (disease);

CREATE TABLE knowledge_chunks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id   UUID        NOT NULL REFERENCES knowledge_documents (id) ON DELETE CASCADE,
    chunk_index   INTEGER     NOT NULL,
    content       TEXT        NOT NULL,
    token_estimate INTEGER    NOT NULL DEFAULT 0,
    page_number   INTEGER,
    section       TEXT,
    metadata_json JSONB       NOT NULL DEFAULT '{}'::jsonb,
    embedding     vector(1024),
    embedding_model TEXT,
    content_tsv   tsvector GENERATED ALWAYS AS (to_tsvector('english', content)) STORED,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT knowledge_chunks_document_index_key UNIQUE (document_id, chunk_index)
);

CREATE INDEX knowledge_chunks_document_id_idx ON knowledge_chunks (document_id);
CREATE INDEX knowledge_chunks_tsv_idx         ON knowledge_chunks USING GIN (content_tsv);

-- Cosine distance index. Rebuild (or switch to HNSW) once the corpus is large;
-- with few rows Postgres will sequential-scan anyway, which is correct.
CREATE INDEX knowledge_chunks_embedding_idx
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);

-- ---------------------------------------------------------------------------
-- AquaDoc conversation domain
-- ---------------------------------------------------------------------------

CREATE TABLE aquadoc_conversations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             TEXT        NOT NULL,
    farm_id             TEXT,
    pond_id             TEXT,
    production_cycle_id TEXT,
    title               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX aquadoc_conversations_user_id_idx ON aquadoc_conversations (user_id);

CREATE TYPE aquadoc_message_role AS ENUM ('user', 'assistant', 'system');

CREATE TABLE aquadoc_messages (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id         UUID        NOT NULL REFERENCES aquadoc_conversations (id) ON DELETE CASCADE,
    role                    aquadoc_message_role NOT NULL,
    content                 TEXT        NOT NULL,
    structured_payload_json JSONB,
    request_id              TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX aquadoc_messages_conversation_id_idx ON aquadoc_messages (conversation_id, created_at);

-- ---------------------------------------------------------------------------
-- Retrieval traces — powers the developer Retrieval Inspector
-- (15_AQUADOC_FRONTEND.md section 4, "Evaluation / Debug") and the provenance
-- record required by 14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 7.
-- ---------------------------------------------------------------------------

CREATE TABLE aquadoc_retrieval_traces (
    request_id      TEXT PRIMARY KEY,
    conversation_id UUID REFERENCES aquadoc_conversations (id) ON DELETE SET NULL,
    question        TEXT        NOT NULL,
    intent          TEXT        NOT NULL,
    trace_json      JSONB       NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX aquadoc_retrieval_traces_created_at_idx ON aquadoc_retrieval_traces (created_at DESC);

-- ---------------------------------------------------------------------------
-- Audit log (07_SECURITY_ARCHITECTURE.md section 11)
-- ---------------------------------------------------------------------------

CREATE TABLE aquadoc_audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_type    TEXT        NOT NULL,
    actor_id      TEXT,
    action        TEXT        NOT NULL,
    resource_type TEXT        NOT NULL,
    resource_id   TEXT,
    before_json   JSONB,
    after_json    JSONB,
    request_id    TEXT,
    ip_address    INET,
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX aquadoc_audit_logs_resource_idx   ON aquadoc_audit_logs (resource_type, resource_id);
CREATE INDEX aquadoc_audit_logs_created_at_idx ON aquadoc_audit_logs (created_at DESC);
