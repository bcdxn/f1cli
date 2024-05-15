package driver

import tea "github.com/charmbracelet/bubbletea"

func fetchDriversStandingsCmd(s teaAppState) tea.Cmd {
	return func() tea.Msg {
		return StandingsMsg{
			standings: s.sc.GetDriversStandings(),
		}
	}
}
