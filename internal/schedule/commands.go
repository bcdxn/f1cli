package schedule

import (
	"errors"

	"github.com/bcdxn/f1cli/internal/f1scraper"
	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/tealogger"
	tea "github.com/charmbracelet/bubbletea"
)

func fetchScheduleCmd() tea.Cmd {
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

func fetchEventDetailsCmd(event *models.RaceEvent) tea.Cmd {
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
