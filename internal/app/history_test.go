package app

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUpsertHistoryMovesExistingTargetToFront(t *testing.T) {
	older := historyEntry{
		repository: repository{Name: "older"},
		Target:     filepath.Join("/tmp", "older"),
		FetchedAt:  time.Now().Add(-time.Hour),
	}
	current := historyEntry{
		repository: repository{Name: "current"},
		Target:     filepath.Join("/tmp", "current"),
	}

	entries := upsertHistory([]historyEntry{older}, current)
	entries = upsertHistory(entries, older)

	if len(entries) != 2 {
		t.Fatalf("upsertHistory() returned %d entries, want 2", len(entries))
	}
	if entries[0].Name != "older" {
		t.Fatalf("upsertHistory() first entry = %q, want older", entries[0].Name)
	}
	if !entries[0].FetchedAt.After(older.FetchedAt) {
		t.Fatal("upsertHistory() did not refresh fetched_at")
	}
}
