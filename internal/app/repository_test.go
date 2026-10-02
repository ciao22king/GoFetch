package app

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		provider string
		repoName string
	}{
		{"github https", "https://github.com/charmbracelet/bubbletea", "GitHub", "bubbletea"},
		{"github https with .git", "https://github.com/owner/project.git", "GitHub", "project"},
		{"github ssh", "git@github.com:ciao22king/GoFetch.git", "GitHub", "GoFetch"},
		{"github ssh scheme", "ssh://git@github.com/owner/project.git", "GitHub", "project"},
		{"gitlab nested group", "gitlab.com/team/tools/clone-kit.git", "GitLab", "clone-kit"},
		{"bare host prefix", "github.com/owner/project", "GitHub", "project"},
		{"self hosted generic", "https://git.example.dev/owner/project", "git.example.dev", "project"},
		{"www prefix stripped", "https://www.github.com/owner/project", "GitHub", "project"},
		{"trailing slash", "https://github.com/owner/project/", "GitHub", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRepository(tt.input)
			if err != nil {
				t.Fatalf("parseRepository() error = %v", err)
			}
			if got.Provider != tt.provider || got.Name != tt.repoName {
				t.Fatalf("parseRepository() = %#v, want provider %q name %q", got, tt.provider, tt.repoName)
			}
		})
	}
}

func TestParseRepositoryRejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", "   "},
		{"spaces", "https://github.com/owner/my project"},
		{"unsupported non domain", "https://localhost/owner/project"},
		{"missing repository", "https://github.com/owner"},
		{"ssh without path", "git@github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseRepository(tt.input); err == nil {
				t.Fatalf("parseRepository(%q) accepted invalid input", tt.input)
			}
		})
	}
}

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"project", "project"},
		{"my:project", "my-project"},
		{"a\\b/c", "a-b-c"},
		{"trailing.", "trailing"},
		{"CON", "CON-repo"},
		{"com1", "com1-repo"},
		{"...", "repository"},
	}
	for _, tt := range tests {
		if got := sanitizeRepoName(tt.input); got != tt.want {
			t.Fatalf("sanitizeRepoName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDestinationPathIsAbsoluteAndExpandsHome(t *testing.T) {
	got, err := destinationPath("~", repository{Name: "project"})
	if err != nil {
		t.Fatalf("destinationPath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("destinationPath() = %q, want an absolute path", got)
	}
	if filepath.Base(got) != "project" {
		t.Fatalf("destinationPath() = %q, want it to end in project", got)
	}
	if runtime.GOOS != "windows" && filepath.Base(filepath.Dir(got)) == "~" {
		t.Fatalf("destinationPath() did not expand the home directory: %q", got)
	}
}

func TestDestinationPathDefaultsToWorkingDirectory(t *testing.T) {
	got, err := destinationPath("", repository{Name: "project"})
	if err != nil {
		t.Fatalf("destinationPath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("destinationPath() = %q, want an absolute path", got)
	}
}
