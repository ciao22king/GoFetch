package app

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type repository struct {
	URL      string
	Provider string
	Name     string
}

// knownProvider maps a canonical host to the provider name shown in the UI.
func knownProvider(host string) (string, bool) {
	switch host {
	case "github.com":
		return "GitHub", true
	case "gitlab.com":
		return "GitLab", true
	default:
		return "", false
	}
}

// genericHosts decides whether an unrecognized host should be accepted as a
// generic Git remote. Obvious non-hosts (localhost, example.*) are rejected so
// typos fail fast instead of turning into a confusing clone error.
func genericHosts(host string) bool {
	switch host {
	case "localhost", "example.com", "example.org", "example.net":
		return false
	default:
		return strings.Contains(host, ".")
	}
}

var unsafeNameChars = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

// sanitizeRepoName turns the repository segment of a URL into a folder name
// that is safe on Windows, macOS and Linux. Windows forbids \ / : * ? " < > |,
// reserves a handful of device names, and disallows trailing dots or spaces;
// those rules are the strictest, so applying them everywhere keeps clones
// portable across platforms.
func sanitizeRepoName(name string) string {
	name = strings.TrimSpace(name)
	name = unsafeNameChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, " .")
	name = strings.TrimRight(name, ".")
	if name == "" {
		return "repository"
	}
	if isReservedWindowsName(name) {
		return name + "-repo"
	}
	return name
}

func isReservedWindowsName(name string) bool {
	base := strings.ToUpper(name)
	if index := strings.IndexByte(base, '.'); index >= 0 {
		base = base[:index]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

// parseRepository understands the URL shapes people actually paste:
//
//	https://github.com/owner/repo(.git)
//	git@github.com:owner/repo(.git)
//	ssh://git@github.com/owner/repo(.git)
//	github.com/owner/repo
//	gitlab.com/group/subgroup/repo
//	file:///path/to/local/repo
//
// GitHub and GitLab are recognized explicitly; any other host that looks like
// a real domain is accepted as a generic Git remote so self-hosted instances
// and other forges keep working. file:// URLs are accepted as local remotes.
func parseRepository(raw string) (repository, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return repository{}, fmt.Errorf("incolla un URL GitHub o GitLab")
	}
	if strings.ContainsAny(raw, " \t\n") {
		return repository{}, fmt.Errorf("l'URL non può contenere spazi")
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
		if parsed.Scheme == "file" {
			// Local repositories need no host validation: file:///path/to/repo
			// is always a valid generic remote, which also makes GoFetch handy
			// for local mirrors and tests.
			name := sanitizeRepoName(strings.TrimSuffix(path.Base(parsed.Path), ".git"))
			if parsed.Path == "" || parsed.Path == "/" {
				return repository{}, fmt.Errorf("URL file:// non valido: percorso mancante")
			}
			return repository{URL: normalized, Provider: "Local", Name: name}, nil
		}
		host = parsed.Hostname()
		repoPath = parsed.Path
	}

	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	if host == "" {
		return repository{}, fmt.Errorf("URL non valido: host mancante")
	}

	provider, known := knownProvider(host)
	if !known {
		if !genericHosts(host) {
			return repository{}, fmt.Errorf("provider non supportato: %s (usa github.com o gitlab.com)", host)
		}
		provider = host
	}

	repoPath = strings.Trim(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	segments := strings.Split(repoPath, "/")
	if len(segments) < 2 || segments[0] == "" || segments[len(segments)-1] == "" {
		return repository{}, fmt.Errorf("serve un URL nel formato %s/owner/repository", host)
	}

	name := sanitizeRepoName(segments[len(segments)-1])
	return repository{URL: normalized, Provider: provider, Name: name}, nil
}

// destinationPath returns an absolute clone target so a later change of the
// process working directory can never move where git actually writes.
func destinationPath(parent string, repo repository) (string, error) {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		parent = "."
	}
	parent = expandHome(parent)
	if strings.HasPrefix(parent, "~/") {
		return "", fmt.Errorf("percorso home non espanso: %s", parent)
	}
	absolute, err := filepath.Abs(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(absolute, repo.Name), nil
}

// nonEmptyDir reports whether path exists and already contains entries, which
// would make git refuse to clone into it.
func nonEmptyDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}
