package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// pullResultMsg reports the outcome of updating an existing clone from the
// history screen.
type pullResultMsg struct {
	target   string
	summary  string
	err      error
	duration time.Duration
}

// pullRepository runs `git pull --ff-only` inside an existing clone. A
// fast-forward-only strategy is deliberate: GoFetch never creates merge
// commits or rebases in a repository it does not own, it only catches the
// clone up with its remote when that is trivially safe.
func pullRepository(target string) tea.Cmd {
	return func() tea.Msg {
		started := time.Now()
		target = expandHome(target)

		info, err := os.Stat(target)
		if err != nil || !info.IsDir() {
			return pullResultMsg{
				target:   target,
				err:      fmt.Errorf("la cartella non esiste più: %s", target),
				duration: time.Since(started),
			}
		}

		cmd := exec.Command("git", "-C", target, "pull", "--ff-only")
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		var output bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &output
		runErr := cmd.Run()

		text := strings.TrimSpace(output.String())
		if runErr != nil {
			return pullResultMsg{
				target:   target,
				err:      friendlyPullError(runErr, text),
				duration: time.Since(started),
			}
		}
		return pullResultMsg{
			target:   target,
			summary:  summarizePull(text),
			duration: time.Since(started),
		}
	}
}

// friendlyPullError condenses git's output into one actionable sentence.
func friendlyPullError(err error, output string) error {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "not a git repository"):
		return fmt.Errorf("la cartella non è un repository Git")
	case strings.Contains(lower, "not possible to fast-forward"),
		strings.Contains(lower, "divergent"):
		return fmt.Errorf("aggiornamento non automatico: il branch ha modifiche divergenti, risolvile con git")
	case strings.Contains(lower, "authentication failed"),
		strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "terminal prompts disabled"):
		return fmt.Errorf("autenticazione richiesta: configura una chiave SSH o un token Git")
	case strings.Contains(lower, "could not resolve host"),
		strings.Contains(lower, "unable to access"),
		strings.Contains(lower, "network is unreachable"):
		return fmt.Errorf("impossibile raggiungere il server: controlla la connessione")
	case strings.Contains(lower, "local changes") ||
		strings.Contains(lower, "would be overwritten"):
		return fmt.Errorf("ci sono modifiche locali: fai commit o stash prima di aggiornare")
	}
	return fmt.Errorf("git pull non riuscito: %s", compactError(err))
}

// summarizePull turns git's success output into a short status line.
func summarizePull(output string) string {
	lower := strings.ToLower(output)
	switch {
	case output == "":
		return "Repository aggiornato."
	case strings.Contains(lower, "already up to date") || strings.Contains(lower, "already up-to-date"):
		return "Già aggiornato: nessuna novità dal remote."
	default:
		// The "Updating <from>..<to>" line is the most informative part of a
		// fast-forward: it says exactly how far the clone moved.
		for _, line := range strings.Split(output, "\n") {
			if line = strings.TrimSpace(line); strings.HasPrefix(strings.ToLower(line), "updating ") {
				return "Aggiornato: " + line
			}
		}
		if line := lastMeaningfulLine(output); line != "" {
			return "Aggiornato: " + line
		}
		return "Repository aggiornato."
	}
}
