package app

import (
	"os"
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

func TestUpsertHistoryCapsEntries(t *testing.T) {
	entries := make([]historyEntry, 0, maxHistoryEntries)
	for i := 0; i < maxHistoryEntries+20; i++ {
		entries = upsertHistory(entries, historyEntry{
			repository: repository{Name: "repo"},
			Target:     filepath.Join("/tmp", "repo", string(rune('a'+i%26))+string(rune('0'+i%10))),
		})
	}
	if len(entries) > maxHistoryEntries {
		t.Fatalf("upsertHistory() kept %d entries, want at most %d", len(entries), maxHistoryEntries)
	}
}

func TestRemoveHistoryEntry(t *testing.T) {
	entries := []historyEntry{
		{Target: filepath.Join("/tmp", "a")},
		{Target: filepath.Join("/tmp", "b")},
	}
	got := removeHistoryEntry(entries, filepath.Join("/tmp", "a"))
	if len(got) != 1 || got[0].Target != filepath.Join("/tmp", "b") {
		t.Fatalf("removeHistoryEntry() = %#v, want only b", got)
	}
}

func TestSearchHistory(t *testing.T) {
	entries := []historyEntry{
		{repository: repository{Name: "bubbletea", Provider: "GitHub", URL: "https://github.com/charmbracelet/bubbletea"}, Target: "/code/bubbletea"},
		{repository: repository{Name: "clone-kit", Provider: "GitLab", URL: "https://gitlab.com/team/clone-kit"}, Target: "/work/clone-kit"},
	}

	if got := searchHistory(entries, ""); len(got) != 2 {
		t.Fatalf("searchHistory(\"\") = %v, want both entries", got)
	}
	if got := searchHistory(entries, "gitlab"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("searchHistory(gitlab) = %v, want [1]", got)
	}
	if got := searchHistory(entries, "/work"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("searchHistory(/work) = %v, want [1]", got)
	}
	if got := searchHistory(entries, "missing"); len(got) != 0 {
		t.Fatalf("searchHistory(missing) = %v, want none", got)
	}
}

func TestRecentDirectories(t *testing.T) {
	entries := []historyEntry{
		{Target: filepath.Join("/code", "a")},
		{Target: filepath.Join("/code", "b")},
		{Target: filepath.Join("/work", "c")},
	}
	got := recentDirectories(entries, 8)
	if len(got) != 2 {
		t.Fatalf("recentDirectories() = %v, want 2 unique directories", got)
	}
}

func TestRelativeTime(t *testing.T) {
	if got := relativeTime(time.Time{}); got != "" {
		t.Fatalf("relativeTime(zero) = %q, want empty", got)
	}
	if got := relativeTime(time.Now()); got != "adesso" {
		t.Fatalf("relativeTime(now) = %q, want adesso", got)
	}
	if got := relativeTime(time.Now().Add(-2 * time.Hour)); got != "2h fa" {
		t.Fatalf("relativeTime(-2h) = %q, want 2h fa", got)
	}
}

func TestSaveAndLoadHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir) // Windows

	entries := []historyEntry{{
		repository: repository{URL: "https://github.com/owner/project", Provider: "GitHub", Name: "project"},
		Target:     filepath.Join(dir, "project"),
		FetchedAt:  time.Now(),
	}}
	if err := saveHistory(entries); err != nil {
		t.Fatalf("saveHistory() error = %v", err)
	}

	loaded := loadHistory()
	if len(loaded) != 1 || loaded[0].Name != "project" {
		t.Fatalf("loadHistory() = %#v, want the saved entry", loaded)
	}

	path, err := historyPath()
	if err != nil {
		t.Fatalf("historyPath() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("history file not written: %v", err)
	}
	// The temp file used for the atomic write must not be left behind.
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "history-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary history files left behind: %v", matches)
	}
}
