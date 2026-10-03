BEGIN;

CREATE TABLE canonical_objects (
    object_kind TEXT NOT NULL CHECK (object_kind IN (
        'installation',
        'identity_user',
        'workspace',
        'project',
        'specification',
        'requirement',
        'release',
        'feature',
        'task',
        'workflow',
        'attempt',
        'change_set',
        'verification',
        'evidence',
        'artifact',
        'decision',
        'approval',
        'risk',
        'change_request',
        'problem',
        'recovery_case',
        'policy',
        'resource',
        'event_audit',
        'operational_knowledge'
    )),
    object_id TEXT NOT NULL CHECK (btrim(object_id) <> ''),
    revision BIGINT NOT NULL CHECK (revision > 0),
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (updated_at >= created_at),
    PRIMARY KEY (object_kind, object_id)
);

CREATE TABLE event_journal (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    object_kind TEXT NOT NULL,
    object_id TEXT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    actor TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    causation_id TEXT,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (object_kind, object_id, revision),
    FOREIGN KEY (object_kind, object_id)
        REFERENCES canonical_objects (object_kind, object_id)
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX event_journal_object_time_idx
    ON event_journal (object_kind, object_id, occurred_at);

CREATE INDEX event_journal_correlation_idx
    ON event_journal (correlation_id, occurred_at);

CREATE TABLE outbox (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE
        REFERENCES event_journal (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    publish_attempts INTEGER NOT NULL DEFAULT 0 CHECK (publish_attempts >= 0),
    CHECK (published_at IS NULL OR published_at >= created_at)
);

CREATE INDEX outbox_pending_idx
    ON outbox (created_at)
    WHERE published_at IS NULL;

CREATE TABLE inbox (
    consumer TEXT NOT NULL,
    message_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    applied_at TIMESTAMPTZ,
    PRIMARY KEY (consumer, message_id),
    UNIQUE (consumer, idempotency_key),
    CHECK (applied_at IS NULL OR applied_at >= received_at)
);

CREATE TABLE side_effect_records (
    idempotency_key TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    target TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    CHECK (completed_at IS NULL OR completed_at >= recorded_at)
);

CREATE OR REPLACE FUNCTION reject_event_journal_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'event_journal is append-only';
END;
$$;

CREATE TRIGGER event_journal_append_only
BEFORE UPDATE OR DELETE ON event_journal
FOR EACH ROW
EXECUTE FUNCTION reject_event_journal_mutation();

COMMIT;
