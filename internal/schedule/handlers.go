package schedule

import (
	"fmt"

	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/tealogger"
	"github.com/charmbracelet/bubbles/list"
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
			Background(lipgloss.Color(f1Red)).
			Foreground(lipgloss.Color("#FFFFFF")).
			PaddingLeft(1)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(f1Red)).
			BorderForeground(lipgloss.Color(f1Red)).
			PaddingLeft(2)
)

// Handlers should return the updated app state and a command (or nil)

// The window size msg handler is the first event fired after Init
func windowSizeMsgHandler(s teaAppState, msg tea.WindowSizeMsg) (teaAppState, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	s.width = msg.Width - h
	s.height = msg.Height - v

	s.hero.SetSize(msg.Width - h - 5)
	tealogger.Log(fmt.Sprintf("window size manager set jhero size %d", msg.Width-h-5))
	if s.isLoading {
		return s, fetchScheduleCmd()
	} else {
		w, h := getListSize(s)
		s.list.SetSize(w, h)
		s.list.Styles.Title = titleStyle.Width(w - 5)
	}
	return s, nil
}

func getListSize(s teaAppState) (int, int) {
	if s.hero.sessions != nil && len(s.hero.sessions) > 0 {
		return s.width, s.height - s.hero.Height()
	} else {
		return s.width, s.height
	}
}

// keyMsgHandler handles key inputs that update the list (e.g. changing the selected item)
func keyMsgHandler(s teaAppState, msg tea.KeyMsg) (teaAppState, tea.Cmd) {
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

// scheduleMsgHandler initializes the list with the F1 schedule data and then returns a tea.Cmd to
// fetch the event details of the 'Hero' event, i.e. the next updcoming or current event
func scheduleMsgHandler(s teaAppState, msg ScheduleMsg) (teaAppState, tea.Cmd) {
	s.schedule = msg.schedule
	s.list = initList(s)
	s.isLoading = false
	return s, fetchEventDetailsCmd(s.schedule.GetHeroEvent())
}

// eventDetailsHandler initializes the schedule 'Hero' event with the event details data.
func eventDetailsMsgHandler(s teaAppState, msg EventDetailsMsg) (teaAppState, tea.Cmd) {
	hero := s.schedule.GetHeroEvent()
	s.schedule.HeroEvent = hero
	s.hero = NewHero(hero.Sessions, s.hero.Width(), 10)
	var cmd tea.Cmd
	s.hero, cmd = s.hero.Update(msg)
	w, h := getListSize(s)
	s.list.SetSize(w, h)
	s.list.Styles.Title = titleStyle.Width(w - 5)
	return s, cmd
}

// the defaultHandler is invoked when no matching event is found
func defaultHandler(s teaAppState, msg tea.Msg) (teaAppState, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// initList customizes and initializes the bubbles list component
func initList(s teaAppState) list.Model {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = selectedStyle.Inherit(d.Styles.SelectedTitle)
	d.Styles.SelectedDesc = selectedStyle.Inherit(d.Styles.SelectedDesc)

	w, h := getListSize(s)
	list := list.New(make([]list.Item, len(s.schedule.Events)), d, w, h)
	list.Title = "Schedule"
	list.SetShowStatusBar(false)
	list.SetShowHelp(false)
	list.Styles.Title = titleStyle.Width(w)

	for i, event := range s.schedule.Events {
		list.SetItem(i, event)
	}

	pos := getInitialCursorPos(s.schedule.Events)

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
