package driver

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// The window size msg handler is the first event fired after Init
func windowSizeMsgHandler(s teaAppState, msg tea.WindowSizeMsg) (teaAppState, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	s.width = msg.Width - h
	s.height = msg.Height - v

	if s.isLoading {
		return s, fetchDriversStandingsCmd(s)
	} else {
		s.table = s.table.Width(s.width)
		s.table = s.table.Height(s.height)
	}
	return s, nil
}

// keyMsgHandler handles key inputs that update the list (e.g. changing the selected item)
func keyMsgHandler(s teaAppState, msg tea.KeyMsg) (teaAppState, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "q", "ctrl+c":
		return s, tea.Quit
	}
	return s, cmd
}

// the defaultHandler is invoked when no matching event is found
func defaultHandler(s teaAppState, msg tea.Msg) (teaAppState, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// standingsMsgHandler initializes the list with the F1 schedule data and then returns a tea.Cmd to
// fetch the event details of the 'Hero' event, i.e. the next updcoming or current event
func standingsMsgHandler(s teaAppState, msg StandingsMsg) (teaAppState, tea.Cmd) {
	s.standings = msg.standings
	s.table = initTable(s)
	s.isLoading = false
	return s, nil
}

func initTable(s teaAppState) *table.Table {
	headers := []string{"POS", "DRIVER", "CONSTRUCTOR", "POINTS"}
	rows := make([][]string, len(s.standings))

	s.l.Debugf("standings length: %d", len(s.standings))
	for i, ds := range s.standings {
		rows[i] = []string{ds.Pos, ds.Name, ds.Constructor, ds.Points}
	}

	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.NormalBorder()).
		Width(s.width).
		Height(s.height).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}

			switch col {
			case 2: // Type 1 + 2
				color := constructorsColors[fmt.Sprint(rows[row-1][col])]
				return baseStyle.Copy().Foreground(color)
			}
			return baseStyle.Copy().Foreground(lipgloss.Color("252"))
		})

	return t
}
