package app

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func saveHistory(entries []historyEntry) error {
	filePath, err := historyPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, append(data, '\n'), 0o644)
}

func upsertHistory(entries []historyEntry, entry historyEntry) []historyEntry {
	filtered := make([]historyEntry, 0, len(entries)+1)
	for _, existing := range entries {
		if filepath.Clean(existing.Target) != filepath.Clean(entry.Target) {
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
