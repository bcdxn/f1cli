package schedule

import (
	"errors"
	"fmt"
	"log"
	"time"

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
			s.list.SetSize(s.width, s.height)
		}
		return s, nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	case ScheduleMsg:
		s.list = initList(msg.events, s.width, s.height)
		s.isLoading = false
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
			titleStyle.Width(s.width).Render("F1 CLI - Schedule"),
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

		events, err := f.GetSchedule()

		if err != nil {
			tealogger.LogErr(err)
			return ErrorMsg(errors.New("error fetching schedule"))
		}

		return ScheduleMsg(ScheduleMsg{
			events: events,
		})
	}
}

func initList(events []models.RaceEvent, width, height int) list.Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(lipgloss.Color(f1Red))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(lipgloss.Color(f1Red))

	list := list.New(make([]list.Item, len(events)), d, width, height-2)
	list.SetShowTitle(false)
	list.SetShowStatusBar(false)

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
func getInitialCursorPos(events []models.RaceEvent) int {
	now := time.Now()

	pos := 0
	for _, item := range events {
		if item.EndsAt.After(now) {
			break
		}
		pos++
	}

	if pos > len(events) {
		pos = 0
	}

	tealogger.Log(fmt.Sprintf("cursor position::::%d", pos))

	return pos
}

type ErrorMsg error
type ScheduleMsg struct {
	events []models.RaceEvent
}

// // create rootmodel.<Model> types that implement required interfaces for bubble components
// type ScheduleModel struct {
// 	events models.RaceEvent
// }

// // implement list.Item interface
// func (s ScheduleModel) FilterValue() string {
// 	return s.events.Location
// }

// func (s ScheduleModel) Title() string {
// 	// // datetime, err := time.Parse("YYYY-MM-DDThh:mm:ssZ07:00", e.StartsAt)
// 	// // start, err := time.Parse("2006-01-02T15:04:05-07:00", rm.e.StartsAt)
// 	// // var (
// 	// // 	startstr string
// 	// // 	end      time.Time
// 	// // 	endstr   string
// 	// // )
// 	// // if err != nil {
// 	// // 	tealogger.LogErr(err)
// 	// // 	startstr = ""
// 	// // 	endstr = ""
// 	// // } else {
// 	// // 	startstr = start.Format("Jan 2")
// 	// // 	end = start.AddDate(0, 0, 3)
// 	// // 	endstr = strconv.Itoa(end.Day())
// 	// // }

// 	// datestr := lipgloss.NewStyle().Faint(true).Render(
// 	// 	// fmt.Sprintf("%s - %s", startstr, endstr),
// 	// 	"1 - 2",
// 	// )

// 	// return fmt.Sprintf("%s %s", datestr, rm.e.Location)
// 	return "test title"
// }

// func (s ScheduleModel) Description() string {
// 	return s.events.OfficialName
// }

// // Main Model
// type Model struct {
// 	schedule list.Model
// 	err      error
// }

// func NewScheduleModel() *Model {
// 	return &Model{}
// }

// // Styling
// // var (
// // 	listStyle = lipgloss.NewStyle().
// // 			Padding(1, 2).
// // 			Border(lipgloss.RoundedBorder()).
// // 			BorderForeground(lipgloss.Color("FF1801"))
// // 	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(241))
// // )

// func (m *Model) initSchedule(width, height int) {
// 	m.schedule = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
// 	m.schedule.Title = "F1 Schedule"
// 	m.schedule.SetItems([]list.Item{})
// }

// func (m Model) Init() tea.Cmd {
// 	// f := f1scraper.New()
// 	return nil
// }

// var (
// 	titleStyle = lipgloss.NewStyle().
// 		Bold(true).
// 		Background(lipgloss.Color("#FF1801")).
// 		Foreground(lipgloss.Color("#FFFFFF"))
// )

// func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	switch msg := msg.(type) {
// 	case tea.WindowSizeMsg:
// 		m.initSchedule(msg.Width-4, msg.Height-4)
// 		titleStyle.Width(msg.Width - 4)
// 	}
// 	var cmd tea.Cmd
// 	m.schedule.SetShowTitle(false)
// 	m.schedule.SetShowStatusBar(false)
// 	m.schedule, cmd = m.schedule.Update(msg)
// 	return m, cmd
// }

// func (m Model) View() string {
// 	return lipgloss.JoinVertical(
// 		lipgloss.Top,
// 		titleStyle.Render("F1 CLI - Schedule"),
// 		m.schedule.View(),
// 	)

// }
