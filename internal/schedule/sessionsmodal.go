package schedule

import (
	"strings"
	"time"

	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/styles"
	"github.com/bcdxn/f1cli/internal/tealogger"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	heroStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(styles.F1Red)).
		Foreground(lipgloss.Color(styles.F1Red)).
		MarginTop(1).
		Padding(1)
)

type heroModel struct {
	gmtOffset string
	sessions  []*models.RaceEventSession
	width     int
	l         tealogger.TeaLogger
}

func NewHero(logger tealogger.TeaLogger, sessions []*models.RaceEventSession, width, height int) heroModel {
	m := heroModel{
		sessions: sessions,
		width:    width,
		l:        logger,
	}

	return m
}

func (m *heroModel) SetSize(width int) {
	m.width = width
}

func (m heroModel) Update(msg tea.Msg) (heroModel, tea.Cmd) {
	switch msg := msg.(type) {
	case EventDetailsMsg:
		m.l.Debugf("SESSIONS:%d", len(msg.sessions))
		m.sessions = msg.sessions
		return m, nil
	default:
		return m, nil
	}
}

func (m heroModel) Height() int {
	return 10
}

func (m heroModel) Width() int {
	return m.width
}

func (m heroModel) View() string {

	str := make([]string, 0, len(m.sessions))
	t := time.Now()
	loc := t.Location()

	for _, session := range m.sessions {
		str = append(str, session.Name+" -- "+session.StartsAt.In(loc).Format("15:04pm MST"))
	}

	return heroStyle.Width(m.width).Render(strings.Join(str, "\n"))
}
