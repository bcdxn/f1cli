package tui

import (
	"fmt"

	"github.com/bcdxn/go-f1/f1livetiming"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type model struct {
	err                string
	isLoadingReference bool
	loadingMsg         string
	spinner            spinner.Model
	c                  f1livetiming.Client
	width              int
	height             int
	done               chan error
	interrupt          chan struct{}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, negotiate(m))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errorMsg:
		return errMsgHandler(m, msg)
	case tea.KeyMsg:
		return keyMsgHandler(m, msg)
	case tea.WindowSizeMsg:
		return windowSizeMsgHandler(m, msg)
	case negotiateMsg:
		return negotiateMsgHandler(m, msg)
	default:
		return defaultHandler(m, msg)
	}
}

func (m model) View() string {
	v := ""

	if m.err != "" {
		v = m.err
	} else if m.isLoadingReference {
		v = fmt.Sprintf(
			"%s %s", m.spinner.View(),
			m.loadingMsg,
		)
	}

	return docStyle.Width(m.width).Render(v)
}

func RunProgram() error {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}

// newModel returns an instance of the tea model needed to start the bubbletea client app
func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	interrupt := make(chan struct{})
	done := make(chan error)

	return model{
		isLoadingReference: true,
		loadingMsg:         "Connecting to F1 LiveTiming...",
		spinner:            s,
		interrupt:          interrupt,
		done:               done,
		c:                  *f1livetiming.NewClient(interrupt, done),
	}
}

/* Tea Messages
------------------------------------------------------------------------------------------------- */

type negotiateMsg struct{}
type connectMsg struct{}
type errorMsg struct {
	err string
}

/* Tea Commands
------------------------------------------------------------------------------------------------- */

func negotiate(m model) tea.Cmd {
	return func() tea.Msg {
		err := m.c.Negotiate()
		if err != nil {
			return errorMsg{
				err: err.Error(),
			}
		}
		return negotiateMsg{}
	}
}

func connect(m model) tea.Cmd {
	return func() tea.Msg {
		go m.c.Connect()
		return connectMsg{}
	}
}

/* Tea Message Handlers
------------------------------------------------------------------------------------------------- */

func defaultHandler(m model, msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func keyMsgHandler(m model, msg tea.KeyMsg) (model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "q", "ctrl+c":
		close(m.interrupt)
		return m, tea.Quit
	}
	return m, cmd
}

func errMsgHandler(m model, msg errorMsg) (model, tea.Cmd) {
	m.err = msg.err
	return m, nil
}

func windowSizeMsgHandler(m model, msg tea.WindowSizeMsg) (model, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	m.width = msg.Width - h
	m.height = msg.Height - v
	return m, nil
}

func negotiateMsgHandler(m model, _ negotiateMsg) (model, tea.Cmd) {
	return m, connect(m)
}
