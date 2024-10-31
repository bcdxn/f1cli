package tui

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/bcdxn/go-f1/f1livetiming"
	"github.com/bcdxn/go-f1/tealogger"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Model struct {
	logger             tealogger.Logger
	err                string
	isLoadingReference bool
	loadingMsg         string
	spinner            spinner.Model
	width              int
	height             int
	done               chan error
	interrupt          chan struct{}
	sessionInfo        f1livetiming.SessionInfo
	driverList         map[string]f1livetiming.DriverData
	timingData         map[string]f1livetiming.DriverTimingData
	lapCount           f1livetiming.LapCount
	table              table.Model
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.logger.Debugf("tea.Msg: %T -- %v", msg, msg)
	switch msg := msg.(type) {
	case ErrorMsg:
		return errMsgHandler(m, msg)
	case tea.KeyMsg:
		return keyMsgHandler(m, msg)
	case tea.WindowSizeMsg:
		return windowSizeMsgHandler(m, msg)
	case SessionInfoMsg:
		return sessionInfoMsgHandler(m, msg)
	case DriverListMsg:
		return driverListMsgHandler(m, msg)
	case LapCountMsg:
		return lapCountMsgHandler(m, msg)
	case TimingDataMsg:
		return timingDataMsgHandler(m, msg)
	case UpdateTableMsg:
		return updateTableMsgHandler(m, msg)
	case DoneMsg:
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		if m.isLoadingReference {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}
}

func (m Model) View() string {
	v := ""

	if m.err != "" {
		v = m.err
	} else if m.isLoadingReference {
		v = fmt.Sprintf(
			"%s %s", m.spinner.View(),
			m.loadingMsg,
		)
	} else {
		v = lipgloss.JoinVertical(
			lipgloss.Top,
			getTitle(m),
			getSubTitle(m),
			m.table.View(),
		)
	}

	return docStyle.Width(m.width).Render(v)
}

// NewModel returns an instance of the tea Model needed to start the bubbletea client app
func NewModel(logger tealogger.Logger, interrupt chan struct{}, done chan error) Model {
	logger.Debug("creating TUI")
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		logger:             logger,
		isLoadingReference: true,
		loadingMsg:         "Connecting to F1 LiveTiming...",
		spinner:            s,
		interrupt:          interrupt,
		done:               done,
		driverList:         make(map[string]f1livetiming.DriverData, 20),
		timingData:         make(map[string]f1livetiming.DriverTimingData, 20),
		table:              newTable(),
	}
}

/* Tea Messages
------------------------------------------------------------------------------------------------- */

type ErrorMsg struct {
	Err error
}

type DoneMsg struct{}

type SessionInfoMsg struct {
	SessionInfo f1livetiming.SessionInfo
}

type DriverListMsg struct {
	DriverList map[string]f1livetiming.DriverData
}

type TimingDataMsg struct {
	TimingData map[string]f1livetiming.DriverTimingData
}

type LapCountMsg struct {
	LapCount f1livetiming.LapCount
}

type UpdateTableMsg struct{}

/* Tea Commands
------------------------------------------------------------------------------------------------- */

func updateTableCmd() tea.Cmd {
	return func() tea.Msg {
		return UpdateTableMsg{}
	}
}

/* Tea Message Handlers
------------------------------------------------------------------------------------------------- */

func keyMsgHandler(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "q", "ctrl+c":
		m.logger.Debug("TUI recevied keymsg and is closing interrupt channel")
		close(m.interrupt)
		return m, cmd
	}
	return m, cmd
}

func errMsgHandler(m Model, msg ErrorMsg) (Model, tea.Cmd) {
	m.err = msg.Err.Error()
	return m, nil
}

func windowSizeMsgHandler(m Model, msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	m.width = msg.Width - h
	m.height = msg.Height - v
	return m, nil
}

func sessionInfoMsgHandler(m Model, msg SessionInfoMsg) (Model, tea.Cmd) {
	m.isLoadingReference = false
	m.sessionInfo = msg.SessionInfo
	return m, updateTableCmd()
}

func driverListMsgHandler(m Model, msg DriverListMsg) (Model, tea.Cmd) {
	for key, driverDelta := range msg.DriverList {
		if driver, ok := m.driverList[key]; ok {
			mergo.Merge(&driver, driverDelta, mergo.WithOverride)
		} else {
			m.driverList[key] = driverDelta
		}
	}
	return m, updateTableCmd()
}

func lapCountMsgHandler(m Model, msg LapCountMsg) (Model, tea.Cmd) {
	m.lapCount.CurrentLap = msg.LapCount.CurrentLap
	if msg.LapCount.TotalLaps > 0 {
		m.lapCount.TotalLaps = msg.LapCount.TotalLaps
	}
	return m, nil
}

func timingDataMsgHandler(m Model, msg TimingDataMsg) (Model, tea.Cmd) {
	for key, newTiming := range msg.TimingData {
		if oldTiming, ok := m.timingData[key]; ok {
			mergo.Merge(&oldTiming, newTiming, mergo.WithOverride)
		} else {
			m.timingData[key] = newTiming
		}
	}
	return m, updateTableCmd()
}

func updateTableMsgHandler(m Model, _ UpdateTableMsg) (Model, tea.Cmd) {
	rows := make([]table.Row, 0, 20)
	for driverNumber, data := range m.driverList {
		lastlap, pBest, oBest := getLastLap(m, driverNumber)

		lastlapStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ebcb8b"))

		if lastlap == "-" {
			lastlapStyle = lipgloss.NewStyle()
		} else if pBest {
			lastlapStyle.Foreground(lipgloss.Color("#a3be8c"))
		} else if oBest {
			lastlapStyle.Foreground(lipgloss.Color("#b48ead"))
		}

		rows = append(rows, table.NewRow(table.RowData{
			"position": data.Line,
			"driver": table.NewStyledCell(
				data.ShortName,
				lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%s", data.TeamColour))),
			),
			"interval": getInterval(m, driverNumber),
			"lastlap":  table.NewStyledCell(lastlap, lastlapStyle),
		}))
	}

	m.table = newTable().WithRows(rows).SortByAsc("position")

	return m, nil
}

/* View Helper Functions
------------------------------------------------------------------------------------------------- */

func getTitle(m Model) string {
	return m.sessionInfo.Meeting.Name
}

func getSubTitle(m Model) string {
	t := m.sessionInfo.Type

	if m.sessionInfo.Type == "Race" {
		t = fmt.Sprintf("%s: %d / %d Laps", t, m.lapCount.CurrentLap, m.lapCount.TotalLaps)
	}

	return t
}

func getInterval(m Model, driverNumber string) string {
	interval := "-"
	if m.timingData[driverNumber].Retired || m.timingData[driverNumber].Status == 4 {
		interval = "DNF"
	} else if m.timingData[driverNumber].IntervalToPositionAhead.Value != "" {
		interval = m.timingData[driverNumber].IntervalToPositionAhead.Value
	}
	return interval
}

func getLastLap(m Model, driverNumber string) (string, bool, bool) {
	var lastlap string
	pBest := m.timingData[driverNumber].LastLapTime.PersonalFastest
	oBest := m.timingData[driverNumber].LastLapTime.OverallFastest
	if m.timingData[driverNumber].Retired || m.timingData[driverNumber].Status == 4 {
		lastlap = "-"
		pBest = false
		oBest = false
	} else if m.timingData[driverNumber].LastLapTime.Value != "" {
		lastlap = m.timingData[driverNumber].LastLapTime.Value
	}

	return lastlap, pBest, oBest

}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

func newTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn("position", "POS", 3),
		table.NewColumn("driver", "DRIVER", 6),
		table.NewColumn("interval", "INT", 8),
		table.NewColumn("lastlap", "LAST", 10),
	}).
		WithRows([]table.Row{}).
		WithBaseStyle(lipgloss.NewStyle().AlignHorizontal(lipgloss.Center))
}
