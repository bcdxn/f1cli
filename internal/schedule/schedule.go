package schedule

import (
	"errors"
	"fmt"
	"log"

	"github.com/bcdxn/f1cli/internal/f1scraper"
	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/tealogger"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	f1Red = "#FF1801"
)

var (
	f1RedText  = lipgloss.NewStyle().Foreground(lipgloss.Color(f1Red))
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#FF1801")).
			Foreground(lipgloss.Color("#FFFFFF"))
)

type teaAppState struct {
	width      int
	height     int
	isQuitting bool
	isLoading  bool
	loadingMsg string
	errMsg     string
	spinner    spinner.Model
	schedule   *models.Schedule
	list       list.Model
}

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
		s.width = msg.Width
		s.height = msg.Height
		if s.isLoading {
			return s, fetchSchedule()
		} else {
			s.list.SetSize(s.width-1, s.height-2)
		}
		return s, nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	case ScheduleMsg:
		s.schedule = msg.schedule
		s.list = initList(s.schedule.Events, s.width-1, s.height-2)
		s.isLoading = false
		// Fetch session details
		return s, fetchEventDetails(s.schedule.GetHeroEvent())
	case EventDetailsMsg:
		tealogger.Log(fmt.Sprintf("sessions:::%d", len(msg.sessions)))
		hero := s.schedule.GetHeroEvent()
		hero.Sessions = msg.sessions
		return s, nil
	default:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
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

func fetchSchedule() tea.Cmd {
	return func() tea.Msg {
		f := f1scraper.New()

		schedule, err := f.GetSchedule()

		if err != nil {
			tealogger.LogErr(err)
			return ErrorMsg(errors.New("error fetching schedule"))
		}

		return ScheduleMsg{
			schedule: schedule,
		}
	}
}

func fetchEventDetails(event *models.RaceEvent) tea.Cmd {
	return func() tea.Msg {
		f := f1scraper.New()
		sessions, err := f.GetEventSessions(event.Location)

		if err != nil {
			tealogger.LogErr(err)
		}

		return EventDetailsMsg{
			sessions: sessions,
		}
	}
}

func initList(events []*models.RaceEvent, width, height int) list.Model {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(lipgloss.Color(f1Red))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(lipgloss.Color(f1Red))

	list := list.New(make([]list.Item, len(events)), d, width, height)
	list.SetShowTitle(false)
	list.SetShowStatusBar(false)
	list.Styles.Spinner = d.Styles.SelectedTitle.Foreground(lipgloss.Color(f1Red))

	for i, event := range events {
		list.SetItem(i, event)
	}

	pos := getInitialCursorPos(events)

	for i := 0; i < pos; i++ {
		list.CursorDown()
	}

	return list
}

// Get index of the first occurence of an event that ends after the current date
func getInitialCursorPos(events []*models.RaceEvent) int {

	pos := 0
	for _, item := range events {
		if item.IsHeroEvent {
			break
		}
		pos++
	}

	if pos > len(events) {
		pos = 0
	}

	return pos
}
