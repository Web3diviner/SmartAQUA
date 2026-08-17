-- Rollback for 0001_initial.sql.
-- Destroys all AquaDoc data. Never run against production without a verified backup.

DROP TABLE IF EXISTS aquadoc_audit_logs;
DROP TABLE IF EXISTS aquadoc_retrieval_traces;
DROP TABLE IF EXISTS aquadoc_messages;
DROP TYPE  IF EXISTS aquadoc_message_role;
DROP TABLE IF EXISTS aquadoc_conversations;
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS knowledge_documents;
DROP TYPE  IF EXISTS knowledge_evidence_level;
DROP TYPE  IF EXISTS knowledge_review_status;
