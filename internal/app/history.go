package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const maxHistoryEntries = 100

type historyEntry struct {
	repository
	Target    string    `json:"target"`
	FetchedAt time.Time `json:"fetched_at"`
}

type openResultMsg struct {
	target string
	err    error
}

func historyPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gofetch", "history.json"), nil
}

func loadHistory() []historyEntry {
	filePath, err := historyPath()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return nil
	}

	var entries []historyEntry
	if json.Unmarshal(data, &entries) != nil {
		return nil
	}
	return entries
}

// saveHistory writes the history atomically: the data is written to a temp file
// in the same directory and renamed over the target. A crash or a full disk can
// then never leave a truncated history.json behind.
func saveHistory(entries []historyEntry) error {
	filePath, err := historyPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	temp, err := os.CreateTemp(dir, "history-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, filePath); err != nil {
		// Windows refuses to rename over an existing file. Fall back to a
		// direct write; it is still better than dropping the update entirely.
		if runtime.GOOS == "windows" {
			return os.WriteFile(filePath, data, 0o644)
		}
		return err
	}
	return nil
}

func upsertHistory(entries []historyEntry, entry historyEntry) []historyEntry {
	filtered := make([]historyEntry, 0, len(entries)+1)
	for _, existing := range entries {
		if !sameTarget(existing.Target, entry.Target) {
			filtered = append(filtered, existing)
		}
	}
	entry.FetchedAt = time.Now()
	filtered = append([]historyEntry{entry}, filtered...)
	if len(filtered) > maxHistoryEntries {
		filtered = filtered[:maxHistoryEntries]
	}
	return filtered
}

func removeHistoryEntry(entries []historyEntry, target string) []historyEntry {
	filtered := make([]historyEntry, 0, len(entries))
	for _, existing := range entries {
		if !sameTarget(existing.Target, target) {
			filtered = append(filtered, existing)
		}
	}
	return filtered
}

func sameTarget(a, b string) bool {
	cleanA := filepath.Clean(expandHome(a))
	cleanB := filepath.Clean(expandHome(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(cleanA, cleanB)
	}
	return cleanA == cleanB
}

func openRepository(target string) tea.Cmd {
	return func() tea.Msg {
		target = expandHome(target)
		var command *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			command = exec.Command("open", target)
		case "windows":
			command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
		default:
			command = exec.Command("xdg-open", target)
		}
		return openResultMsg{target: target, err: command.Start()}
	}
}

func historyLabel(entry historyEntry) string {
	name := entry.Name
	if name == "" {
		name = filepath.Base(strings.TrimRight(entry.Target, string(os.PathSeparator)))
	}
	return name
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	delta := time.Since(t)
	switch {
	case delta < time.Minute:
		return "adesso"
	case delta < time.Hour:
		return fmt.Sprintf("%dm fa", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh fa", int(delta.Hours()))
	case delta < 30*24*time.Hour:
		return fmt.Sprintf("%dg fa", int(delta.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// searchHistory returns the indexes of the entries matching query, preserving
// the newest-first order. An empty query matches everything.
func searchHistory(entries []historyEntry, query string) []int {
	query = strings.ToLower(strings.TrimSpace(query))
	indexes := make([]int, 0, len(entries))
	for index, entry := range entries {
		if query == "" || historyMatches(entry, query) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func historyMatches(entry historyEntry, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		entry.Name,
		entry.Provider,
		entry.Target,
		entry.URL,
	}, " "))
	return strings.Contains(haystack, query)
}

// recentDirectories returns the distinct parent directories seen in the
// history, alphabetically sorted, to power destination suggestions.
func recentDirectories(entries []historyEntry, limit int) []string {
	seen := make(map[string]struct{}, len(entries))
	directories := make([]string, 0, limit)
	for _, entry := range entries {
		if entry.Target == "" {
			continue
		}
		dir := filepath.Dir(expandHome(entry.Target))
		key := dir
		if runtime.GOOS == "windows" {
			key = strings.ToLower(dir)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		directories = append(directories, dir)
	}
	sort.Strings(directories)
	if limit > 0 && len(directories) > limit {
		directories = directories[:limit]
	}
	return directories
}

func expandHome(value string) string {
	value = strings.TrimSpace(value)
	if value == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return value
	}
	if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, value[2:])
		}
	}
	return value
}
