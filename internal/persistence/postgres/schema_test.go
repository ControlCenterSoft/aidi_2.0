package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestCanonicalStateMigrationContainsMandatoryInvariants(t *testing.T) {
	raw, err := os.ReadFile("migrations/0001_canonical_state.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)

	required := []string{
		"CREATE TABLE canonical_objects",
		"PRIMARY KEY (object_kind, object_id)",
		"CREATE TABLE event_journal",
		"UNIQUE (object_kind, object_id, revision)",
		"schema_version INTEGER NOT NULL CHECK (schema_version > 0)",
		"CREATE TABLE outbox",
		"event_id TEXT NOT NULL UNIQUE",
		"WHERE published_at IS NULL",
		"CREATE TABLE inbox",
		"PRIMARY KEY (consumer, message_id)",
		"UNIQUE (consumer, idempotency_key)",
		"CREATE TABLE side_effect_records",
		"idempotency_key TEXT PRIMARY KEY",
		"CREATE TRIGGER event_journal_append_only",
		"BEFORE UPDATE OR DELETE ON event_journal",
	}

	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing invariant %q", fragment)
		}
	}
}

func TestCanonicalStateRollbackRemovesAllObjects(t *testing.T) {
	raw, err := os.ReadFile("migrations/0001_canonical_state.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)

	required := []string{
		"DROP TRIGGER IF EXISTS event_journal_append_only",
		"DROP FUNCTION IF EXISTS reject_event_journal_mutation",
		"DROP TABLE IF EXISTS side_effect_records",
		"DROP TABLE IF EXISTS inbox",
		"DROP TABLE IF EXISTS outbox",
		"DROP TABLE IF EXISTS event_journal",
		"DROP TABLE IF EXISTS canonical_objects",
	}

	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("rollback missing %q", fragment)
		}
	}
}
