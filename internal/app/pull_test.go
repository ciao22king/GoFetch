package app

import (
	"errors"
	"strings"
	"testing"
)

func TestSummarizePull(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"empty", "", "Repository aggiornato."},
		{"up to date", "Already up to date.", "Già aggiornato: nessuna novità dal remote."},
		{"up-to-date variant", "Already up-to-date.", "Già aggiornato: nessuna novità dal remote."},
		{"updated", "Updating cfcd826..50d2057\nFast-forward\n main.go | 10 +++++-----", "Aggiornato: Updating cfcd826..50d2057"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarizePull(tt.output); got != tt.want {
				t.Errorf("summarizePull(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestFriendlyPullError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"not a repo", "fatal: not a git repository", "non è un repository Git"},
		{"diverged", "fatal: Not possible to fast-forward, aborting.", "modifiche divergenti"},
		{"auth", "fatal: Authentication failed", "autenticazione richiesta"},
		{"network", "fatal: unable to access 'https://x': Could not resolve host", "controlla la connessione"},
		{"local changes", "error: Your local changes to the following files would be overwritten by merge", "modifiche locali"},
		{"fallback", "something unexpected", "git pull non riuscito"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := friendlyPullError(errors.New("exit status 1"), tt.output)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("friendlyPullError(%q) = %v, want substring %q", tt.output, err, tt.want)
			}
		})
	}
}
