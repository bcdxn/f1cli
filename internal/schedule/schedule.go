package schedule

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Create a new application state object (called models in bubbletea)
func newTeaAppState() teaAppState {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = f1RedText

	return teaAppState{
		width:      0,
		height:     0,
		isLoading:  true,
		isQuitting: false,
		loadingMsg: "Retrieving F1 schedule...",
		spinner:    s,
	}
}

// The first hook called by bubbletea; we'll start a spinner
func (s teaAppState) Init() tea.Cmd {
	if s.isLoading {
		return s.spinner.Tick
	}
	return nil
}

// Update the application state appropriately based on the message
func (s teaAppState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		return windowSizeMsgHandler(s, msg)
	case tea.KeyMsg:
		return keyMsgHandler(s, msg)
	case ScheduleMsg:
		return scheduleMsgHandler(s, msg)
	case EventDetailsMsg:
		return eventDetailsMsgHandler(s, msg)
	default:
		return defaultHandler(s, msg)
	}
}

// Render the view
func (s teaAppState) View() string {
	str := ""
	if s.isLoading {
		str = fmt.Sprintf(
			"%s %s", s.spinner.View(),
			f1RedText.Render(s.loadingMsg),
			// lipgloss.NewStyle().Foreground(lipgloss.Color(f1Red)).Render(s.loadingMsg),
		)
	} else {
		str = lipgloss.JoinVertical(
			lipgloss.Top,
			"",
			titleStyle.Width(s.width).Render("Schedule"),
			s.list.View(),
		)
	}

	if s.isQuitting {
		return str + "\n"
	}

	return str
}

// Run the bubbletea program
func RunProgram() {
	p := tea.NewProgram(newTeaAppState())

	_, err := p.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
