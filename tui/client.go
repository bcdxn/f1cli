package tui

import (
	"fmt"

	"github.com/bcdxn/go-f1/f1livetiming"
	"github.com/bcdxn/go-f1/tealogger"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Model struct {
	logger             tealogger.Logger
	err                string
	isLoadingReference bool
	loadingMsg         string
	spinner            spinner.Model
	width              int
	height             int
	done               chan error
	interrupt          chan struct{}
	sessionInfo        f1livetiming.SessionInfo
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.logger.Debugf("tea.Msg: %T", msg)
	switch msg := msg.(type) {
	case ErrorMsg:
		return errMsgHandler(m, msg)
	case tea.KeyMsg:
		return keyMsgHandler(m, msg)
	case tea.WindowSizeMsg:
		return windowSizeMsgHandler(m, msg)
	case DoneMsg:
		return m, tea.Quit
	case SessionInfoMsg:
		m.isLoadingReference = false
		m.sessionInfo = msg.SessionInfo
		return m, nil
	default:
		return defaultHandler(m, msg)
	}
}

func (m Model) View() string {
	v := ""

	if m.err != "" {
		v = m.err
	} else if m.isLoadingReference {
		v = fmt.Sprintf(
			"%s %s", m.spinner.View(),
			m.loadingMsg,
		)
	} else {
		v = m.sessionInfo.Meeting.Name
	}

	return docStyle.Width(m.width).Render(v)
}

// NewModel returns an instance of the tea Model needed to start the bubbletea client app
func NewModel(logger tealogger.Logger, interrupt chan struct{}, done chan error) Model {
	logger.Debug("creating TUI")
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		logger:             logger,
		isLoadingReference: true,
		loadingMsg:         "Connecting to F1 LiveTiming...",
		spinner:            s,
		interrupt:          interrupt,
		done:               done,
	}
}

/* Tea Messages
------------------------------------------------------------------------------------------------- */

type ErrorMsg struct {
	Err error
}

type DoneMsg struct{}

type SessionInfoMsg struct {
	SessionInfo f1livetiming.SessionInfo
}

/* Tea Commands
------------------------------------------------------------------------------------------------- */

/* Tea Message Handlers
------------------------------------------------------------------------------------------------- */

func defaultHandler(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func keyMsgHandler(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "q", "ctrl+c":
		m.logger.Debug("TUI recevied keymsg and is closing interrupt channel")
		close(m.interrupt)
		return m, cmd
	}
	return m, cmd
}

func errMsgHandler(m Model, msg ErrorMsg) (Model, tea.Cmd) {
	m.err = msg.Err.Error()
	return m, nil
}

func windowSizeMsgHandler(m Model, msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	m.width = msg.Width - h
	m.height = msg.Height - v
	return m, nil
}
