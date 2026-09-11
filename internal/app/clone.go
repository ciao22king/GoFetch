package app

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type cloneResultMsg struct {
	target   string
	output   string
	err      error
	duration time.Duration
}

func cloneRepository(repo repository, parent string) tea.Cmd {
	return func() tea.Msg {
		started := time.Now()
		parent = expandHome(parent)
		target := filepath.Join(parent, repo.Name)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return cloneResultMsg{target: target, err: err, duration: time.Since(started)}
		}

		ctx := context.Background()
		cmd := exec.CommandContext(ctx, "git", "clone", "--progress", repo.URL, target)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		var output bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()

		return cloneResultMsg{
			target:   target,
			output:   strings.TrimSpace(output.String()),
			err:      err,
			duration: time.Since(started),
		}
	}
}

func expandHome(value string) string {
	if value == "~" || strings.HasPrefix(value, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	return value
}
