package tui

import (
	"context"
	"log/slog"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/bcdxn/f1cli/internal/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	s = styles.Default()
)

func NewLeaderboard(opts ...TUIOption) *tea.Program {
	l := Leaderboard{
		drivers: make(map[string]domain.Driver),
		logger:  slog.Default(),
		ctx:     context.Background(),
	}
	// apply given options
	for _, opt := range opts {
		opt(&l)
	}
	// return new Bubbletea program
	return tea.NewProgram(l, tea.WithContext(l.ctx))
}

type TUIOption = func(c *Leaderboard)

// WithLogger configures the logger to use within the TUI program
func WithLogger(l *slog.Logger) TUIOption {
	return func(b *Leaderboard) { b.logger = l }
}

// WithLogger configures the context to use within the TUI program
func WithContext(ctx context.Context) TUIOption {
	return func(b *Leaderboard) { b.ctx = ctx }
}

/* Bubbletea Interface Implementation
------------------------------------------------------------------------------------------------- */

func (l Leaderboard) Init() tea.Cmd {
	return nil
}

func (l Leaderboard) View() string {
	return "running leaderboard"
}

func (l Leaderboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return handleKeyMsg(l, msg)
	}
	return l, nil
}

/* Tea Mesage Types
------------------------------------------------------------------------------------------------- */

type DriversMsg map[string]domain.Driver
type MeetingMsg domain.Meeting
type RaceCtrlMsgsMsg []domain.RaceCtrlMsg

/* Tea Mesage handlers
------------------------------------------------------------------------------------------------- */

func handleKeyMsg(m Leaderboard, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.logger.Debug("received quit tea message")
		return m, tea.Quit
	}
	return m, nil
}

/* Type Definitions
------------------------------------------------------------------------------------------------- */

type Leaderboard struct {
	drivers map[string]domain.Driver
	logger  *slog.Logger
	ctx     context.Context
}
