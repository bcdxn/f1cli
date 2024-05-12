package schedule

import (
	"github.com/bcdxn/f1cli/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	heroStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(f1Red))
)

type heroModel struct {
	sessions []*models.RaceEventSession
	width    int
}

func NewHero(sessions []*models.RaceEventSession, width, height int) heroModel {
	m := heroModel{
		sessions: sessions,
		width:    width - 2,
	}

	return m
}

func (m *heroModel) SetSize(width, height int) {
	m.width = width
}

func (m heroModel) Update(msg tea.Msg) (heroModel, tea.Cmd) {
	switch msg := msg.(type) {
	case EventDetailsMsg:
		m.sessions = msg.sessions
		return m, nil
	default:
		return m, nil
	}
}

func (m heroModel) View() string {
	return heroStyle.Width(m.width).Render("::::::::hero:::::::::")
}
