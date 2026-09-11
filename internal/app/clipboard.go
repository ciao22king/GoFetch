package app

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type clipboardMsg struct {
	value string
	err   error
}

func readClipboard() tea.Cmd {
	return func() tea.Msg {
		value, err := clipboard.ReadAll()
		return clipboardMsg{value: value, err: err}
	}
}
