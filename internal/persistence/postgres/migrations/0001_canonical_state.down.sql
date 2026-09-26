BEGIN;

DROP TRIGGER IF EXISTS event_journal_append_only ON event_journal;
DROP FUNCTION IF EXISTS reject_event_journal_mutation();

DROP TABLE IF EXISTS side_effect_records;
DROP TABLE IF EXISTS inbox;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS event_journal;
DROP TABLE IF EXISTS canonical_objects;

COMMIT;
