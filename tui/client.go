package tui

import (
	"github.com/bcdxn/go-f1/f1livetiming"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	prop string
	c    f1livetiming.Client
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return keyMsgHandler(m, msg)
	default:
		return defaultHandler(m, msg)
	}
}

func (m model) View() string {
	return m.prop
}

func RunProgram() error {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}

/* Tea Message Handlers
------------------------------------------------------------------------------------------------- */

func defaultHandler(m model, _ tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func keyMsgHandler(m model, msg tea.KeyMsg) (model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, cmd
}
