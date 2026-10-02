package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type cloneResultMsg struct {
	target   string
	output   string
	err      error
	duration time.Duration
	build    buildResult
}

// cloneRepository runs `git clone` in a cancellable goroutine. The cancel func
// is stored on the model so the cloning screen can abort a stuck network fetch
// (for example a private repository waiting on credentials) with Esc. When
// overwrite is true, an existing non-empty destination is removed first.
// noBuild skips the automatic build step entirely.
func cloneRepository(repo repository, parent string, overwrite, noBuild bool) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := func() tea.Msg {
		started := time.Now()
		target, err := destinationPath(parent, repo)
		if err != nil {
			return cloneResultMsg{target: target, err: err, duration: time.Since(started)}
		}
		if nonEmptyDir(target) {
			if !overwrite {
				return cloneResultMsg{
					target:   target,
					err:      fmt.Errorf("la cartella %s esiste già e non è vuota", target),
					duration: time.Since(started),
				}
			}
			if err := os.RemoveAll(target); err != nil {
				return cloneResultMsg{target: target, err: err, duration: time.Since(started)}
			}
		}
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return cloneResultMsg{target: target, err: err, duration: time.Since(started)}
		}

		gitCmd := exec.CommandContext(ctx, "git", "clone", "--progress", repo.URL, target)
		gitCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		var output bytes.Buffer
		gitCmd.Stdout = &output
		gitCmd.Stderr = &output
		runErr := gitCmd.Run()

		if ctx.Err() == context.Canceled {
			// git leaves a half-written directory behind on cancel; remove it
			// so the next attempt starts from a clean state.
			_ = os.RemoveAll(target)
			return cloneResultMsg{target: target, err: errCloneCanceled, duration: time.Since(started)}
		}

		build := buildResult{skipped: true}
		if runErr == nil && !noBuild {
			build = buildRepository(target)
		}

		return cloneResultMsg{
			target:   target,
			output:   strings.TrimSpace(output.String()),
			err:      friendlyCloneError(runErr, output.String(), target),
			duration: time.Since(started),
			build:    build,
		}
	}
	return cmd, cancel
}

var errCloneCanceled = fmt.Errorf("clone annullato")

// friendlyCloneError replaces git's wall of text with a short, actionable hint
// while preserving the raw output for anyone who needs to debug it.
func friendlyCloneError(err error, output, target string) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "repository not found"):
		return fmt.Errorf("repository non trovato o privato: controlla l'URL e le tue credenziali")
	case strings.Contains(lower, "authentication failed"),
		strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "could not read from remote repository"),
		strings.Contains(lower, "terminal prompts disabled"):
		return fmt.Errorf("autenticazione richiesta: configura una chiave SSH o un token Git")
	case strings.Contains(lower, "could not resolve host"),
		strings.Contains(lower, "unable to access"),
		strings.Contains(lower, "network is unreachable"):
		return fmt.Errorf("impossibile raggiungere il server: controlla la connessione")
	case strings.Contains(lower, "already exists and is not an empty directory"):
		return fmt.Errorf("la cartella %s esiste già e non è vuota", target)
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fmt.Errorf("git clone non riuscito")
	}
	return fmt.Errorf("git clone non riuscito: %s", message)
}
