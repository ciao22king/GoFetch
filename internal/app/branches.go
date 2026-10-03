package app

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// branchesMsg delivers the branch names advertised by the remote, fetched in
// the background while the user is on the branch screen.
type branchesMsg struct {
	url      string
	branches []string
	err      error
}

// fetchRemoteBranches lists the remote's branches without cloning anything:
// `git ls-remote --heads` only reads refs, so it is fast and needs no working
// tree. The URL is echoed back in the message so a stale response (the user
// went back and picked another repository) can be discarded.
func fetchRemoteBranches(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", url)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.Output()
		if err != nil {
			return branchesMsg{url: url, err: err}
		}
		return branchesMsg{url: url, branches: parseRemoteBranches(string(output))}
	}
}

// parseRemoteBranches extracts branch names from `git ls-remote --heads`
// output, whose lines look like:
//
//	65049fb8fccea09cc947147362a92031dffd9bf8	refs/heads/main
func parseRemoteBranches(output string) []string {
	var branches []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if name, ok := strings.CutPrefix(fields[1], "refs/heads/"); ok && name != "" {
			branches = append(branches, name)
		}
	}
	return branches
}

// branchCompletions returns the remote branches matching the typed prefix,
// with exact-prefix matches first. An empty prefix returns everything, so Tab
// on an empty field cycles through all branches.
func branchCompletions(branches []string, prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	matches := make([]string, 0, len(branches))
	for _, branch := range branches {
		if prefix == "" || strings.HasPrefix(branch, prefix) {
			matches = append(matches, branch)
		}
	}
	return matches
}
