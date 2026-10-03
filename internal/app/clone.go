package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
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

// progressMsg carries a single line of `git clone --progress` output to the
// cloning screen so the user can watch the fetch happen instead of a frozen
// spinner.
type progressMsg struct{ line string }

// progressTickMsg is delivered on a fixed interval while a clone is running so
// the elapsed timer in the cloning view keeps counting even when git is quiet.
type progressTickMsg struct{}

const cloneProgressInterval = time.Second

// maxProgressLines bounds how much clone output the UI keeps in memory and
// shows; older lines scroll out of view.
const maxProgressLines = 200

func progressTick() tea.Cmd {
	return tea.Tick(cloneProgressInterval, func(time.Time) tea.Msg { return progressTickMsg{} })
}

// cloneRepository runs `git clone` in a cancellable goroutine and streams its
// progress to the TUI. The cancel func is stored on the model so the cloning
// screen can abort a stuck network fetch (for example a private repository
// waiting on credentials) with Esc. When overwrite is true, an existing
// non-empty destination is removed first. An empty branch clones the remote's
// default branch. A depth greater than zero performs a shallow clone limited
// to that many commits, which is much faster on large repositories. noBuild
// skips the automatic build step entirely.
func cloneRepository(repo repository, parent, branch string, overwrite, noBuild bool, depth int, progress chan<- string) (tea.Cmd, context.CancelFunc) {
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

		args := []string{"clone", "--progress"}
		if strings.TrimSpace(branch) != "" {
			args = append(args, "--branch", branch)
		}
		if depth > 0 {
			args = append(args, "--depth", fmt.Sprintf("%d", depth))
		}
		args = append(args, repo.URL, target)

		gitCmd := exec.CommandContext(ctx, "git", args...)
		gitCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		var combined bytes.Buffer
		reader, writer := io.Pipe()
		gitCmd.Stdout = writer
		gitCmd.Stderr = writer

		// git writes clone progress to stderr. Drain the pipe on a separate
		// goroutine so a slow UI can never block git, and publish each line both
		// to the TUI (progress) and to a buffer for the final result.
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(reader)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				combined.WriteString(line)
				combined.WriteByte('\n')
				if progress != nil {
					select {
					case progress <- line:
					default:
					}
				}
			}
		}()

		runErr := gitCmd.Run()
		writer.Close()
		wg.Wait()

		if ctx.Err() == context.Canceled {
			// git leaves a half-written directory behind on cancel; remove it
			// so the next attempt starts from a clean state.
			_ = os.RemoveAll(target)
			return cloneResultMsg{target: target, err: errCloneCanceled, duration: time.Since(started)}
		}

		output := combined.String()
		build := buildResult{skipped: true}
		if runErr == nil && !noBuild {
			build = buildRepository(target)
		}

		return cloneResultMsg{
			target:   target,
			output:   strings.TrimSpace(output),
			err:      friendlyCloneError(runErr, output, target),
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
	case strings.Contains(lower, "remote branch") && strings.Contains(lower, "not found"):
		return fmt.Errorf("branch remoto inesistente: controlla il nome del branch")
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
