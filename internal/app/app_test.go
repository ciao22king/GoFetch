package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(t *testing.T, opts Options) model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AppData", t.TempDir()) // Windows config location
	if opts.Version == "" {
		opts.Version = "test"
	}
	return newModel(opts)
}

func apply(t *testing.T, m model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("Update() returned %T, want model", updated)
	}
	return next, cmd
}

func TestFlowURLToDestination(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	if m.screen != urlScreen {
		t.Fatalf("initial screen = %v, want urlScreen", m.screen)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != destinationScreen {
		t.Fatalf("screen after URL = %v, want destinationScreen", m.screen)
	}
	if m.repository.Name != "project" {
		t.Fatalf("parsed repository name = %q, want project", m.repository.Name)
	}
}

func TestInvalidURLKeepsUserOnURLScreen(t *testing.T) {
	m := newTestModel(t, Options{})
	m.screen = urlScreen
	m.urlInput.SetValue("https://github.com/owner")
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != urlScreen {
		t.Fatalf("screen = %v, want urlScreen after invalid URL", m.screen)
	}
	if m.err == nil {
		t.Fatal("expected a validation error for an incomplete URL")
	}
}

func TestQuitKeyIgnoredWhileTyping(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, cmd := apply(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil || m.quitting {
		t.Fatal("pressing q while typing a URL must not quit")
	}
}

func TestOverwriteConfirmation(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "project")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "existing"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.dirInput.SetValue(parent)

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.confirmOverwrite {
		t.Fatal("expected an overwrite confirmation for a non-empty destination")
	}
	if m.screen != destinationScreen {
		t.Fatalf("screen = %v, want destinationScreen while confirming", m.screen)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirmOverwrite {
		t.Fatal("Esc must cancel the overwrite confirmation")
	}
}

func TestEscCancelsClone(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.dirInput.SetValue(t.TempDir())

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != cloningScreen {
		t.Fatalf("screen = %v, want cloningScreen", m.screen)
	}
	if m.cancelClone == nil {
		t.Fatal("expected a cancel function to be stored while cloning")
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.cancelClone == nil {
		t.Fatal("cancel function should remain until the clone result arrives")
	}
}

func TestCloneSuccessStoresHistory(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = apply(t, m, cloneResultMsg{target: filepath.Join(t.TempDir(), "project")})
	if m.screen != historyScreen {
		t.Fatalf("screen = %v, want historyScreen after success", m.screen)
	}
	if len(m.history) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.history))
	}
}

func TestCloneFailureShowsResultScreen(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = apply(t, m, cloneResultMsg{err: errCloneCanceled})
	if m.screen != resultScreen {
		t.Fatalf("screen = %v, want resultScreen after failure", m.screen)
	}
}

func TestHistoryNavigationAndSearch(t *testing.T) {
	m := newTestModel(t, Options{})
	m.history = []historyEntry{
		{repository: repository{Name: "alpha", Provider: "GitHub"}, Target: "/code/alpha"},
		{repository: repository{Name: "beta", Provider: "GitLab"}, Target: "/work/beta"},
	}
	m.refreshFilter()

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 1 {
		t.Fatalf("selected = %d, want 1 after down", m.selected)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	if !m.searchActive {
		t.Fatal("ctrl+l should activate the search field")
	}
	for _, r := range "beta" {
		m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(m.filtered) != 1 || m.history[m.filtered[0]].Name != "beta" {
		t.Fatalf("filtered = %v, want only beta", m.filtered)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchActive {
		t.Fatal("Esc should close the search field")
	}
}

func TestHistoryRemoveEntry(t *testing.T) {
	m := newTestModel(t, Options{})
	m.history = []historyEntry{
		{repository: repository{Name: "alpha"}, Target: "/code/alpha"},
	}
	m.refreshFilter()
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(m.history) != 0 {
		t.Fatalf("history length = %d, want 0 after ctrl+d", len(m.history))
	}
}

func TestTabCyclesDestinationSuggestions(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m.history = []historyEntry{
		{Target: filepath.Join("/code", "alpha")},
		{Target: filepath.Join("/work", "beta")},
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyTab})
	first := m.dirInput.Value()
	if first == "" {
		t.Fatal("tab did not produce a destination suggestion")
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.dirInput.Value() == first {
		t.Fatal("tab did not cycle to a different suggestion")
	}
}
