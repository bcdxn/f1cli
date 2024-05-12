package schedule

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

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
		loadingMsg: f1RedText.Render("Retrieving F1 schedule..."),
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
		str = docStyle.Render(fmt.Sprintf(
			"%s %s", s.spinner.View(),
			s.loadingMsg,
		))
	} else {
		str = docStyle.Render(s.list.View())
	}
	// else {
	// 	str = lipgloss.JoinVertical(
	// 		lipgloss.Top,
	// 		"",
	// 		titleStyle.Width(s.width).Render("Schedule"),
	// 		s.list.View(),
	// 		s.hero.View(),
	// 	)
	// }

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
