package app

import "testing"

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		provider string
		repoName string
	}{
		{"github https", "https://github.com/charmbracelet/bubbletea", "GitHub", "bubbletea"},
		{"github ssh", "git@github.com:ciao22king/GoFetch.git", "GitHub", "GoFetch"},
		{"gitlab nested group", "gitlab.com/team/tools/clone-kit.git", "GitLab", "clone-kit"},
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

func TestParseRepositoryRejectsUnsupportedHost(t *testing.T) {
	if _, err := parseRepository("https://codeberg.org/team/project"); err == nil {
		t.Fatal("parseRepository() accepted unsupported host")
	}
}
