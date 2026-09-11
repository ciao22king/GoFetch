package app

import (
	"fmt"
	"strings"
	"time"

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
	screen     screen
	urlInput   textinput.Model
	dirInput   textinput.Model
	spinner    spinner.Model
	repository repository
	result     cloneResultMsg
	history    []historyEntry
	selected   int
	err        error
	status     string
	width      int
	height     int
	version    string
	startedAt  time.Time
	initialURL string
	quitting   bool
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
	quit  key.Binding
	enter key.Binding
	back  key.Binding
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
}

func Run(initialURL, initialDir, version string) error {
	program := tea.NewProgram(newModel(initialURL, initialDir, version), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(initialURL, initialDir, version string) model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://github.com/owner/repository"
	urlInput.Prompt = "  "
	urlInput.CharLimit = 500
	urlInput.Width = 60
	urlInput.SetValue(initialURL)

	dirInput := textinput.New()
	dirInput.Placeholder = "~/Code"
	dirInput.Prompt = "  "
	dirInput.CharLimit = 500
	dirInput.Width = 60
	dirInput.SetValue(initialDir)

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(cyan)

	m := model{
		urlInput:   urlInput,
		dirInput:   dirInput,
		spinner:    spin,
		version:    version,
		initialURL: initialURL,
		history:    loadHistory(),
	}
	if initialURL != "" {
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.urlInput.Width = min(70, max(35, msg.Width-16))
		m.dirInput.Width = m.urlInput.Width
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
	case tea.KeyMsg:
		if key.Matches(msg, appKeys.quit) && m.screen != cloningScreen {
			m.quitting = true
			return m, tea.Quit
		}
		if (m.screen == urlScreen || m.screen == destinationScreen) &&
			(msg.Type == tea.KeyCtrlV || msg.String() == "ctrl+v") {
			return m, readClipboard()
		}
	case cloneResultMsg:
		m.result = msg
		if msg.err == nil {
			entry := historyEntry{
				repository: m.repository,
				Target:     msg.target,
			}
			m.history = upsertHistory(m.history, entry)
			if err := saveHistory(m.history); err != nil {
				m.status = "Clone completato, ma non riesco a salvare la cronologia."
			} else {
				m.status = "Clone completato e aggiunto ai fetch recenti."
			}
			m.selected = 0
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
	}

	switch m.screen {
	case historyScreen:
		return m.updateHistory(msg)
	case urlScreen:
		return m.updateURL(msg)
	case destinationScreen:
		return m.updateDestination(msg)
	case cloningScreen:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case resultScreen:
		return m.updateResult(msg)
	}
	return m, nil
}

func (m model) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		if len(m.history) > 0 {
			m.selected = max(0, m.selected-1)
		}
	case "down", "j":
		if len(m.history) > 0 {
			m.selected = min(len(m.history)-1, m.selected+1)
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
		m.selected = min(m.selected, max(len(m.history)-1, 0))
		m.status = "Cronologia aggiornata."
	case "enter":
		if len(m.history) > 0 {
			m.status = "Apro " + historyLabel(m.history[m.selected]) + "…"
			return m, openRepository(m.history[m.selected].Target)
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
		if key.Matches(msg, appKeys.back) {
			m.screen = urlScreen
			m.dirInput.Blur()
			m.urlInput.Focus()
			return m, textinput.Blink
		}
		if key.Matches(msg, appKeys.enter) {
			m.err = nil
			m.status = ""
			m.screen = cloningScreen
			m.startedAt = time.Now()
			m.dirInput.Blur()
			return m, tea.Batch(m.spinner.Tick, cloneRepository(m.repository, m.dirInput.Value()))
		}
	}
	var cmd tea.Cmd
	m.dirInput, cmd = m.dirInput.Update(msg)
	return m, cmd
}

func (m model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, appKeys.enter):
			m.screen = historyScreen
			m.err = nil
			return m, textinput.Blink
		case key.Matches(keyMsg, appKeys.back):
			m.screen = destinationScreen
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
	if len(m.history) == 0 {
		body = cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("Nessun fetch ancora"),
			"",
			subtitleStyle.Render("Premi n per clonare il tuo primo repository."),
		))
	} else {
		start := max(0, m.selected-4)
		end := min(len(m.history), start+8)
		if end-start < 8 {
			start = max(0, end-8)
		}
		rows := make([]string, 0, end-start)
		for index, entry := range m.history[start:end] {
			rows = append(rows, m.viewHistoryRow(start+index, entry))
		}
		body = cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	status := ""
	if m.status != "" {
		status = subtitleStyle.Render(m.status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, copy, "", body, status)
}

func (m model) viewHistoryRow(index int, entry historyEntry) string {
	prefix := "  "
	nameStyle := lipgloss.NewStyle().Foreground(ink)
	if index == m.selected {
		prefix = "› "
		nameStyle = nameStyle.Foreground(cyan).Bold(true)
	}
	name := nameStyle.Render(historyLabel(entry))
	provider := lipgloss.NewStyle().Foreground(muted).Render(entry.Provider)
	target := lipgloss.NewStyle().Foreground(muted).Render(entry.Target)
	return lipgloss.JoinVertical(lipgloss.Left,
		prefix+name+"  "+provider,
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
	return lipgloss.JoinVertical(lipgloss.Left, title, copy, "", field, m.viewStatus())
}

func (m model) viewCloning() string {
	elapsed := time.Since(m.startedAt).Round(time.Second)
	title := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("Fetching your repository")
	target := subtitleStyle.Render(m.repository.Provider + "  ·  " + m.repository.Name)
	status := lipgloss.NewStyle().Foreground(cyan).Render(m.spinner.View() + "  git is doing its thing…")
	timer := helpStyle.Render(fmt.Sprintf("elapsed %s", elapsed))
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
	timer := helpStyle.Render(fmt.Sprintf("finished in %s", m.result.duration.Round(time.Second)))
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
		return helpStyle.Render("↑/↓ scegli   •   enter apri   •   n nuovo fetch   •   q esci")
	case urlScreen:
		return helpStyle.Render("ctrl+v incolla   •   enter continua   •   esc indietro   •   q esci")
	case destinationScreen:
		return helpStyle.Render("ctrl+v incolla   •   enter clona   •   esc indietro   •   q esci")
	case cloningScreen:
		return helpStyle.Render("attendi   •   git clone in corso")
	case resultScreen:
		return helpStyle.Render("enter torna ai fetch   •   esc indietro   •   q esci")
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
