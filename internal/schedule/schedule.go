package schedule

import (
	"fmt"
	"log"

	"github.com/bcdxn/f1cli/internal/f1scraper"
	"github.com/bcdxn/f1cli/internal/tealogger"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// Create a new application state object (called models in bubbletea)
func newTeaAppState(o ScheduleOptions, sc f1scraper.F1ScraperClient, l tealogger.TeaLogger) teaAppState {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = f1RedText

	return teaAppState{
		width:             0,
		height:            0,
		isLoading:         true,
		isQuitting:        false,
		loadingMsg:        f1RedText.Render("Retrieving F1 schedule..."),
		spinner:           s,
		displayTrackTimes: o.DisplayTrackTimes,
		sc:                sc,
		l:                 l,
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
	} else if s.hero.sessions != nil && len(s.hero.sessions) > 0 {
		str = docStyle.Render(lipgloss.JoinVertical(
			lipgloss.Top,
			s.list.View(),
			s.hero.View(),
		))
	} else {
		str = docStyle.Render(s.list.View())
	}

	if s.isQuitting {
		return str + "\n\n"
	}

	return str
}

type ScheduleOptions struct {
	DisplayTrackTimes bool
	Debug             bool
}

// Run the bubbletea program
func RunProgram(o ScheduleOptions) {
	l := tealogger.New(o.Debug)
	f := f1scraper.New(l)
	l.Debug("running schedule program")
	p := tea.NewProgram(newTeaAppState(o, *f, l), tea.WithAltScreen())

	_, err := p.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
