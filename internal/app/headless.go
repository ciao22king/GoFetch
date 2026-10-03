package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunHeadless clones a repository without the TUI: it is what powers
// `gofetch -url <url> -yes`, so the tool can be used inside scripts and CI
// pipelines where no terminal interaction is possible. Progress goes to
// stderr and the final destination path to stdout, so the path can be
// captured with command substitution.
func RunHeadless(opts Options) error {
	repo, err := parseRepository(opts.InitialURL)
	if err != nil {
		return err
	}

	parent := opts.InitialDir
	if parent == "" {
		var wdErr error
		if parent, wdErr = os.Getwd(); wdErr != nil {
			return wdErr
		}
	}

	target, err := destinationPath(parent, repo)
	if err != nil {
		return err
	}
	if nonEmptyDir(target) {
		if !opts.Overwrite {
			return fmt.Errorf("la cartella %s esiste già e non è vuota (usa -force per sostituirla)", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	branch := strings.TrimSpace(opts.InitialBranch)
	fmt.Fprintf(os.Stderr, "Clono %s (%s) in %s…\n", repo.Name, repo.Provider, target)

	started := time.Now()
	if err := headlessClone(repo, branch, opts.Depth, target); err != nil {
		return err
	}
	elapsed := time.Since(started).Round(time.Second)

	if !opts.NoBuild {
		build := buildRepository(target)
		switch {
		case build.skipped:
			fmt.Fprintln(os.Stderr, "Nessuna build riconosciuta: clone completato.")
		case build.err != nil:
			fmt.Fprintf(os.Stderr, "Attenzione: build fallita (%s): %s\n", build.command, compactError(build.err))
		default:
			fmt.Fprintf(os.Stderr, "Build completata (%s) in %s.\n", build.command, build.duration.Round(time.Second))
		}
	}

	entry := historyEntry{repository: repo, Target: target}
	_ = saveHistory(upsertHistory(loadHistory(), entry))

	fmt.Fprintf(os.Stderr, "Fatto in %s.\n", elapsed)
	fmt.Println(target)
	return nil
}

// headlessClone runs git clone with progress streamed straight to stderr.
func headlessClone(repo repository, branch string, depth int, target string) error {
	args := []string{"clone", "--progress"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	args = append(args, repo.URL, target)

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	var combined strings.Builder
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			combined.WriteString(line)
			combined.WriteByte('\n')
			fmt.Fprintln(os.Stderr, line)
		}
	}()

	runErr := cmd.Run()
	writer.Close()
	<-done
	return friendlyCloneError(runErr, combined.String(), target)
}
