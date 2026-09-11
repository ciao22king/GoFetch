package app

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

type repository struct {
	URL      string
	Provider string
	Name     string
}

func parseRepository(raw string) (repository, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return repository{}, fmt.Errorf("incolla un URL GitHub o GitLab")
	}

	normalized := raw
	if !strings.Contains(normalized, "://") && !strings.HasPrefix(normalized, "git@") {
		normalized = "https://" + normalized
	}

	var host, repoPath string
	switch {
	case strings.HasPrefix(normalized, "git@"):
		// SCP-like SSH URLs: git@github.com:owner/repository.git
		parts := strings.SplitN(normalized, ":", 2)
		if len(parts) != 2 {
			return repository{}, fmt.Errorf("URL SSH non valido")
		}
		host = strings.TrimPrefix(parts[0], "git@")
		repoPath = parts[1]
	default:
		parsed, err := url.Parse(normalized)
		if err != nil {
			return repository{}, fmt.Errorf("URL non valido: %w", err)
		}
		host = strings.ToLower(parsed.Hostname())
		repoPath = parsed.Path
	}

	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	provider := ""
	switch host {
	case "github.com":
		provider = "GitHub"
	case "gitlab.com":
		provider = "GitLab"
	default:
		return repository{}, fmt.Errorf("provider non supportato: %s (usa github.com o gitlab.com)", host)
	}

	repoPath = strings.Trim(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	segments := strings.Split(repoPath, "/")
	if len(segments) < 2 || segments[0] == "" || segments[len(segments)-1] == "" {
		return repository{}, fmt.Errorf("serve un URL nel formato %s/owner/repository", host)
	}

	name := segments[len(segments)-1]
	return repository{URL: normalized, Provider: provider, Name: name}, nil
}

func destinationPath(parent string, repo repository) (string, error) {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		parent = "."
	}
	if strings.HasPrefix(parent, "~/") {
		// Keep this helper independent from the user's shell. The caller
		// expands the home directory before invoking git.
		return "", fmt.Errorf("percorso home non espanso: %s", parent)
	}
	return path.Join(parent, repo.Name), nil
}
