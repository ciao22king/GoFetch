package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	historyScreen screen = iota
	urlScreen
	destinationScreen
	cloningScreen
	resultScreen
)

type model struct {
	screen       screen
	urlInput     textinput.Model
	dirInput     textinput.Model
	searchInput  textinput.Model
	spinner      spinner.Model
	repository   repository
	result       cloneResultMsg
	history      []historyEntry
	filtered     []int
	selected     int
	err          error
	status       string
	width        int
	height       int
	version      string
	startedAt    time.Time
	initialURL   string
	quitting     bool
	searchActive bool

	confirmOverwrite bool
	suggestionIndex  int
	cancelClone      context.CancelFunc
	noBuild          bool
}

var (
	primary = lipgloss.Color("#8B5CF6")
	cyan    = lipgloss.Color("#22D3EE")
	green   = lipgloss.Color("#34D399")
	muted   = lipgloss.Color("#94A3B8")
	ink     = lipgloss.Color("#E2E8F0")
	danger  = lipgloss.Color("#FB7185")

	logoStyle = lipgloss.NewStyle().
			Foreground(ink).
			Bold(true).
			Background(primary).
			Padding(0, 1)
	subtitleStyle = lipgloss.NewStyle().Foreground(muted)
	labelStyle    = lipgloss.NewStyle().Foreground(muted).Bold(true)
	cardStyle     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#334155")).
			Padding(1, 2)
	helpStyle = lipgloss.NewStyle().Foreground(muted)
)

type keys struct {
	quit   key.Binding
	enter  key.Binding
	back   key.Binding
	open   key.Binding
	remove key.Binding
	copy   key.Binding
	search key.Binding
}

var appKeys = keys{
	quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	),
	back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	open: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open folder"),
	),
	remove: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "remove"),
	),
	copy: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy path"),
	),
	search: key.NewBinding(
		key.WithKeys("ctrl+l", "/"),
		key.WithHelp("ctrl+l", "search"),
	),
}

type copiedMsg struct{ err error }

// Options configures a GoFetch run.
type Options struct {
	InitialURL string
	InitialDir string
	Version    string
	NoBuild    bool
}

func Run(opts Options) error {
	program := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(opts Options) model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://github.com/owner/repository"
	urlInput.Prompt = "  "
	urlInput.CharLimit = 500
	urlInput.Width = 60
	urlInput.SetValue(opts.InitialURL)

	dirInput := textinput.New()
	dirInput.Placeholder = "~/Code"
	dirInput.Prompt = "  "
	dirInput.CharLimit = 500
	dirInput.Width = 60
	dirInput.SetValue(opts.InitialDir)

	searchInput := textinput.New()
	searchInput.Placeholder = "cerca per nome, provider o percorso"
	searchInput.Prompt = "  / "
	searchInput.CharLimit = 200
	searchInput.Width = 60

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(cyan)

	m := model{
		urlInput:    urlInput,
		dirInput:    dirInput,
		searchInput: searchInput,
		spinner:     spin,
		version:     opts.Version,
		initialURL:  opts.InitialURL,
		history:     loadHistory(),
		noBuild:     opts.NoBuild,
	}
	m.refreshFilter()
	if opts.InitialURL != "" {
		m.screen = urlScreen
		urlInput.Focus()
	} else {
		m.screen = historyScreen
		urlInput.Blur()
	}
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// refreshFilter recomputes the visible history indexes and keeps the selection
// inside bounds. It is called whenever the history or the search query changes.
func (m *model) refreshFilter() {
	m.filtered = searchHistory(m.history, m.searchInput.Value())
	if len(m.filtered) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.urlInput.Width = min(70, max(35, msg.Width-16))
		m.dirInput.Width = m.urlInput.Width
		m.searchInput.Width = m.urlInput.Width
		return m, nil
	case clipboardMsg:
		if msg.err != nil {
			m.status = "Clipboard non disponibile: " + compactError(msg.err)
			return m, nil
		}
		if strings.TrimSpace(msg.value) == "" {
			m.status = "La clipboard è vuota."
			return m, nil
		}
		switch m.screen {
		case urlScreen:
			m.urlInput.SetValue(strings.TrimSpace(msg.value))
			m.status = "URL incollato dalla clipboard."
		case destinationScreen:
			m.dirInput.SetValue(strings.TrimSpace(msg.value))
			m.status = "Percorso incollato dalla clipboard."
		}
		return m, nil
	case copiedMsg:
		if msg.err != nil {
			m.status = "Copia non riuscita: " + compactError(msg.err)
		} else {
			m.status = "Percorso copiato nella clipboard."
		}
		return m, nil
	case cloneResultMsg:
		m.cancelClone = nil
		m.result = msg
		if msg.err == nil {
			entry := historyEntry{
				repository: m.repository,
				Target:     msg.target,
			}
			m.history = upsertHistory(m.history, entry)
			m.selected = 0
			m.refreshFilter()
			if err := saveHistory(m.history); err != nil {
				m.status = "Clone completato, ma non riesco a salvare la cronologia."
			} else if msg.build.command != "" && msg.build.err != nil {
				m.status = fmt.Sprintf("Clone completato, ma la build è fallita (%s): %s", msg.build.command, compactError(msg.build.err))
			} else if msg.build.command != "" {
				m.status = fmt.Sprintf("Clone e build completati (%s).", msg.build.command)
			} else {
				m.status = "Clone completato e aggiunto ai fetch recenti. Nessuna build riconosciuta."
			}
			m.screen = historyScreen
		} else {
			m.screen = resultScreen
		}
		return m, nil
	case openResultMsg:
		if msg.err != nil {
			m.status = "Non riesco ad aprire la cartella: " + compactError(msg.err)
		} else {
			m.status = "Cartella aperta: " + msg.target
		}
		return m, nil
	case tea.KeyMsg:
		if cmd, handled := m.handleGlobalKey(msg); handled {
			return m, cmd
		}
	}

	switch m.screen {
	case historyScreen:
		return m.updateHistory(msg)
	case urlScreen:
		return m.updateURL(msg)
	case destinationScreen:
		return m.updateDestination(msg)
	case cloningScreen:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, appKeys.back) {
			if m.cancelClone != nil {
				m.cancelClone()
				m.status = "Annullamento del clone…"
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case resultScreen:
		return m.updateResult(msg)
	}
	return m, nil
}

// handleGlobalKey covers shortcuts that behave the same everywhere. Typing
// shortcuts like "q" are deliberately excluded from text-entry screens so a URL
// or a path containing the letter q can never quit the app.
func (m model) handleGlobalKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if msg.Type == tea.KeyCtrlC {
		if m.cancelClone != nil {
			m.cancelClone()
		}
		m.quitting = true
		return tea.Quit, true
	}
	if (m.screen == urlScreen || m.screen == destinationScreen) && msg.Type == tea.KeyCtrlV {
		return readClipboard(), true
	}
	if key.Matches(msg, appKeys.open) {
		if m.screen == resultScreen && m.result.target != "" {
			return openRepository(m.result.target), true
		}
		if m.screen == historyScreen && len(m.filtered) > 0 {
			entry := m.history[m.filtered[m.selected]]
			m.status = "Apro " + historyLabel(entry) + "…"
			return openRepository(entry.Target), true
		}
		return nil, true
	}
	if key.Matches(msg, appKeys.copy) {
		if m.screen == resultScreen && m.result.target != "" {
			return copyPath(m.result.target), true
		}
		if m.screen == historyScreen && len(m.filtered) > 0 {
			return copyPath(m.history[m.filtered[m.selected]].Target), true
		}
		return nil, true
	}
	if key.Matches(msg, appKeys.quit) {
		typing := (m.screen == historyScreen && m.searchActive) ||
			m.screen == urlScreen || m.screen == destinationScreen
		if !typing {
			m.quitting = true
			return tea.Quit, true
		}
	}
	return nil, false
}

func (m model) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.searchActive {
		switch keyMsg.Type {
		case tea.KeyEsc:
			m.searchActive = false
			m.searchInput.Blur()
			return m, nil
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				entry := m.history[m.filtered[m.selected]]
				m.searchActive = false
				m.searchInput.Blur()
				m.status = "Apro " + historyLabel(entry) + "…"
				return m, openRepository(entry.Target)
			}
			return m, nil
		case tea.KeyUp:
			if len(m.filtered) > 0 {
				m.selected = max(0, m.selected-1)
			}
			return m, nil
		case tea.KeyDown:
			if len(m.filtered) > 0 {
				m.selected = min(len(m.filtered)-1, m.selected+1)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.selected = 0
		m.refreshFilter()
		return m, cmd
	}

	switch keyMsg.String() {
	case "up", "k":
		if len(m.filtered) > 0 {
			m.selected = max(0, m.selected-1)
		}
	case "down", "j":
		if len(m.filtered) > 0 {
			m.selected = min(len(m.filtered)-1, m.selected+1)
		}
	case "n", "c":
		m.status = ""
		m.err = nil
		m.screen = urlScreen
		m.urlInput.SetValue("")
		m.urlInput.Focus()
		return m, textinput.Blink
	case "r":
		m.history = loadHistory()
		m.refreshFilter()
		m.status = "Cronologia aggiornata."
	case "ctrl+l", "/":
		m.searchActive = true
		m.status = ""
		m.searchInput.Focus()
		return m, textinput.Blink
	case "ctrl+d":
		if len(m.filtered) > 0 {
			entry := m.history[m.filtered[m.selected]]
			m.history = removeHistoryEntry(m.history, entry.Target)
			m.refreshFilter()
			if err := saveHistory(m.history); err != nil {
				m.status = "Voce rimossa, ma non riesco a salvare la cronologia."
			} else {
				m.status = "Voce rimossa: " + historyLabel(entry)
			}
		}
	case "enter":
		if len(m.filtered) > 0 {
			entry := m.history[m.filtered[m.selected]]
			m.status = "Apro " + historyLabel(entry) + "…"
			return m, openRepository(entry.Target)
		}
	}
	return m, nil
}

func (m model) updateURL(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, appKeys.back) {
			m.screen = historyScreen
			m.urlInput.Blur()
			m.status = ""
			m.err = nil
			return m, nil
		}
		if key.Matches(msg, appKeys.enter) {
			repo, err := parseRepository(m.urlInput.Value())
			if err != nil {
				m.err = err
				return m, nil
			}
			m.repository = repo
			m.err = nil
			m.status = ""
			m.screen = destinationScreen
			m.urlInput.Blur()
			m.dirInput.Focus()
			m.suggestionIndex = -1
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.urlInput, cmd = m.urlInput.Update(msg)
	return m, cmd
}

func (m model) updateDestination(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmOverwrite {
			switch {
			case key.Matches(msg, appKeys.enter):
				m.confirmOverwrite = false
				return m.startClone(true)
			case key.Matches(msg, appKeys.back):
				m.confirmOverwrite = false
				m.status = "Sovrascrittura annullata."
			}
			return m, nil
		}
		if key.Matches(msg, appKeys.back) {
			m.screen = urlScreen
			m.dirInput.Blur()
			m.urlInput.Focus()
			m.err = nil
			return m, textinput.Blink
		}
		if key.Matches(msg, appKeys.enter) {
			target, err := destinationPath(m.dirInput.Value(), m.repository)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			if nonEmptyDir(target) {
				m.confirmOverwrite = true
				return m, nil
			}
			return m.startClone(false)
		}
		if msg.Type == tea.KeyTab {
			if suggestion := m.nextSuggestion(); suggestion != "" {
				m.dirInput.SetValue(suggestion)
				m.dirInput.CursorEnd()
				m.status = "Suggerimento: " + suggestion
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.dirInput, cmd = m.dirInput.Update(msg)
	return m, cmd
}

func (m model) startClone(overwrite bool) (tea.Model, tea.Cmd) {
	m.status = ""
	m.screen = cloningScreen
	m.startedAt = time.Now()
	m.dirInput.Blur()
	cmd, cancel := cloneRepository(m.repository, m.dirInput.Value(), overwrite, m.noBuild)
	m.cancelClone = cancel
	return m, tea.Batch(m.spinner.Tick, cmd)
}

// nextSuggestion cycles through the most recent destination directories. It
// uses a pointer receiver so the cursor advances across calls.
func (m *model) nextSuggestion() string {
	suggestions := recentDirectories(m.history, 8)
	if len(suggestions) == 0 {
		return ""
	}
	m.suggestionIndex = (m.suggestionIndex + 1) % len(suggestions)
	return suggestions[m.suggestionIndex]
}

func (m model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, appKeys.enter):
			if m.result.err == nil && m.result.target != "" {
				return m, openRepository(m.result.target)
			}
			m.screen = historyScreen
			m.err = nil
			return m, textinput.Blink
		case key.Matches(keyMsg, appKeys.back):
			m.screen = destinationScreen
			m.confirmOverwrite = false
			m.dirInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "starting gofetch..."
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		logoStyle.Render("GOFETCH"),
		"  ",
		subtitleStyle.Render("clone without the ceremony"),
		"  ",
		helpStyle.Render("v"+m.version),
	)
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", m.viewBody(), "", m.viewFooter())
	maxWidth := min(max(m.width-4, 40), 90)
	return lipgloss.NewStyle().Width(maxWidth).MarginLeft(2).MarginTop(1).Render(content)
}

func (m model) viewBody() string {
	switch m.screen {
	case historyScreen:
		return m.viewHistory()
	case urlScreen:
		return m.viewURL()
	case destinationScreen:
		return m.viewDestination()
	case cloningScreen:
		return m.viewCloning()
	case resultScreen:
		return m.viewResult()
	default:
		return ""
	}
}

func (m model) viewHistory() string {
	title := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("Your fetches")
	copy := subtitleStyle.Render("Scegli un repository con ↑/↓ e premi Invio per aprire la cartella.")

	var body string
	switch {
	case len(m.history) == 0:
		body = cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("Nessun fetch ancora"),
			"",
			subtitleStyle.Render("Premi n per clonare il tuo primo repository."),
		))
	case len(m.filtered) == 0:
		body = cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("Nessun risultato"),
			"",
			subtitleStyle.Render("Nessun fetch corrisponde a \""+m.searchInput.Value()+"\"."),
		))
	default:
		start := max(0, m.selected-4)
		end := min(len(m.filtered), start+8)
		if end-start < 8 {
			start = max(0, end-8)
		}
		rows := make([]string, 0, end-start)
		for position := start; position < end; position++ {
			rows = append(rows, m.viewHistoryRow(position, m.history[m.filtered[position]]))
		}
		body = cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, copy, "", m.viewSearchBar(), "", body, m.viewStatus())
}

func (m model) viewSearchBar() string {
	switch {
	case m.searchActive:
		return cardStyle.Render(m.searchInput.View())
	case strings.TrimSpace(m.searchInput.Value()) != "":
		return helpStyle.Render("  filtro attivo: \"" + m.searchInput.Value() + "\"   •   ctrl+l per modificare")
	default:
		return helpStyle.Render("  / oppure ctrl+l per cercare nella cronologia")
	}
}

func (m model) viewHistoryRow(position int, entry historyEntry) string {
	prefix := "  "
	nameStyle := lipgloss.NewStyle().Foreground(ink)
	if position == m.selected {
		prefix = "› "
		nameStyle = nameStyle.Foreground(cyan).Bold(true)
	}
	name := nameStyle.Render(historyLabel(entry))
	provider := lipgloss.NewStyle().Foreground(muted).Render(entry.Provider)
	when := lipgloss.NewStyle().Foreground(muted).Render(relativeTime(entry.FetchedAt))
	target := lipgloss.NewStyle().Foreground(muted).Render(entry.Target)
	return lipgloss.JoinVertical(lipgloss.Left,
		prefix+name+"  "+provider+"  "+when,
		"   "+target,
	)
}

func (m model) viewURL() string {
	title := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("Where are we going?")
	copy := subtitleStyle.Render("Paste a GitHub or GitLab URL. HTTPS and SSH both work.")
	field := cardStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("REPOSITORY URL"),
			m.urlInput.View(),
			"",
			providerPill("GITHUB", "github.com")+"  "+providerPill("GITLAB", "gitlab.com"),
		),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, copy, "", field, m.viewError(), m.viewStatus())
}

func (m model) viewDestination() string {
	title := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("Where should we put it?")
	copy := subtitleStyle.Render("GoFetch creates a folder named after the repository.")
	repoBadge := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render(m.repository.Provider + "  ·  " + m.repository.Name)
	field := cardStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			repoBadge,
			"",
			labelStyle.Render("PARENT DIRECTORY"),
			m.dirInput.View(),
		),
	)

	extra := ""
	if m.confirmOverwrite {
		extra = "\n" + lipgloss.NewStyle().Foreground(danger).Bold(true).
			Render("  La cartella esiste già: Invio la sostituisce, Esc annulla.")
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, copy, "", field, m.viewError(), m.viewStatus()+extra)
}

func (m model) viewCloning() string {
	elapsed := time.Since(m.startedAt).Round(time.Second)
	title := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("Fetching your repository")
	target := subtitleStyle.Render(m.repository.Provider + "  ·  " + m.repository.Name)
	status := lipgloss.NewStyle().Foreground(cyan).Render(m.spinner.View() + "  clone + build automatici in corso…")
	timer := helpStyle.Render(fmt.Sprintf("elapsed %s   •   esc per annullare", elapsed))
	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", target, status, timer))
}

func (m model) viewResult() string {
	if m.result.err != nil {
		title := lipgloss.NewStyle().Foreground(danger).Bold(true).Render("Clone failed")
		detail := subtitleStyle.Render(compactError(m.result.err))
		help := helpStyle.Render("Check the URL, your SSH key, or your Git credentials.")
		return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", detail, help))
	}
	title := lipgloss.NewStyle().Foreground(green).Bold(true).Render("Clone complete")
	target := lipgloss.NewStyle().Foreground(ink).Render(m.result.target)
	timer := helpStyle.Render(fmt.Sprintf("finished in %s   •   invio apre la cartella", m.result.duration.Round(time.Second)))
	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", target, timer))
}

func (m model) viewError() string {
	if m.err == nil {
		return ""
	}
	return lipgloss.NewStyle().Foreground(danger).Render("  " + m.err.Error())
}

func (m model) viewStatus() string {
	if m.status == "" {
		return ""
	}
	return helpStyle.Render("  " + m.status)
}

func (m model) viewFooter() string {
	switch m.screen {
	case historyScreen:
		return helpStyle.Render("↑/↓ scegli   •   enter apri   •   n nuovo   •   / cerca   •   ctrl+d rimuovi   •   q esci")
	case urlScreen:
		return helpStyle.Render("ctrl+v incolla   •   enter continua   •   esc indietro   •   ctrl+c esci")
	case destinationScreen:
		return helpStyle.Render("ctrl+v incolla   •   tab suggerimento   •   enter clona   •   esc indietro")
	case cloningScreen:
		return helpStyle.Render("attendi   •   esc annulla")
	case resultScreen:
		return helpStyle.Render("enter apri   •   ctrl+y copia percorso   •   esc indietro   •   q esci")
	default:
		return ""
	}
}

func providerPill(label, value string) string {
	return lipgloss.NewStyle().
		Foreground(muted).
		Background(lipgloss.Color("#1E293B")).
		Padding(0, 1).
		Render(label + "  " + value)
}

func copyPath(target string) tea.Cmd {
	return func() tea.Msg {
		return copiedMsg{err: clipboard.WriteAll(target)}
	}
}

func compactError(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) > 160 {
		return message[:157] + "..."
	}
	return message
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
