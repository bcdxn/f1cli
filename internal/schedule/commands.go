package schedule

import (
	tea "github.com/charmbracelet/bubbletea"
)

func fetchScheduleCmd(s teaAppState) tea.Cmd {
	return func() tea.Msg {
		return ScheduleMsg{
			schedule: s.sc.GetSchedule(),
		}
	}
}

func fetchEventDetailsCmd(s teaAppState) tea.Cmd {
	return func() tea.Msg {
		sessions, err := s.sc.GetEventSessions(s.schedule.GetHeroEvent().EventDetailLink)

		if err != nil {
			s.l.LogErr(err)
		}

		return EventDetailsMsg{
			sessions: sessions,
		}
	}
}
