package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFriendlyCloneError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"not found", "remote: Repository not found.", "non trovato"},
		{"auth", "fatal: Authentication failed for 'https://example.com'", "autenticazione"},
		{"dns", "fatal: unable to access: Could not resolve host: github.com", "raggiungere"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := friendlyCloneError(errCloneCanceled, tt.output, "/tmp/x")
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.want) {
				t.Fatalf("friendlyCloneError() = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestFriendlyCloneErrorNil(t *testing.T) {
	if err := friendlyCloneError(nil, "anything", "/tmp/x"); err != nil {
		t.Fatalf("friendlyCloneError(nil) = %v, want nil", err)
	}
}

func TestNonEmptyDir(t *testing.T) {
	empty := t.TempDir()
	if nonEmptyDir(empty) {
		t.Fatalf("nonEmptyDir() = true for an empty directory")
	}

	populated := t.TempDir()
	if err := os.WriteFile(filepath.Join(populated, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !nonEmptyDir(populated) {
		t.Fatalf("nonEmptyDir() = false for a populated directory")
	}

	if nonEmptyDir(filepath.Join(empty, "missing")) {
		t.Fatalf("nonEmptyDir() = true for a missing directory")
	}
}

func TestCloneRepositoryEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	origin := t.TempDir()
	runGit(t, origin, "init", "-q")
	runGit(t, origin, "config", "user.email", "test@example.com")
	runGit(t, origin, "config", "user.name", "GoFetch Test")
	if err := os.WriteFile(filepath.Join(origin, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, origin, "add", ".")
	runGit(t, origin, "commit", "-q", "-m", "init")

	parent := t.TempDir()
	repo := repository{URL: "file://" + origin, Provider: "GitHub", Name: "cloned"}
	cmd, cancel := cloneRepository(repo, parent, false, true)
	defer cancel()

	msg := cmd()
	result, ok := msg.(cloneResultMsg)
	if !ok {
		t.Fatalf("cloneRepository() returned %T, want cloneResultMsg", msg)
	}
	if result.err != nil {
		t.Fatalf("cloneRepository() error = %v", result.err)
	}
	if _, err := os.Stat(filepath.Join(parent, "cloned", "README.md")); err != nil {
		t.Fatalf("cloned repository missing README: %v", err)
	}

	// A second clone into the same target must be refused without overwrite.
	cmd2, cancel2 := cloneRepository(repo, parent, false, true)
	defer cancel2()
	msg2 := cmd2().(cloneResultMsg)
	if msg2.err == nil {
		t.Fatal("cloneRepository() overwrote a non-empty target without permission")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
