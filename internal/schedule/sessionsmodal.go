package schedule

import (
	"fmt"
	"strings"

	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/tealogger"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	heroStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(f1Red)).
		Foreground(lipgloss.Color(f1Red)).
		MarginTop(1).
		Padding(1)
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
	h, _ := docStyle.GetFrameSize()
	m.width = width - h
}

func (m heroModel) Update(msg tea.Msg) (heroModel, tea.Cmd) {
	switch msg := msg.(type) {
	case EventDetailsMsg:
		tealogger.Log(fmt.Sprintf("SESSIONS:%d", len(msg.sessions)))
		m.sessions = msg.sessions
		return m, nil
	default:
		return m, nil
	}
}

func (m heroModel) Height() int {
	return 10
}

func (m heroModel) View() string {

	str := make([]string, 0, len(m.sessions))

	for _, session := range m.sessions {
		str = append(str, session.Name+" "+session.StartsAt.Format("15:04 MST"))
	}

	return heroStyle.Render(strings.Join(str, "\n"))
}
