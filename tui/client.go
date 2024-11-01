package tui

import (
	"fmt"
	"strconv"

	"dario.cat/mergo"
	"github.com/bcdxn/go-f1/f1livetiming"
	"github.com/bcdxn/go-f1/tealogger"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

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
	lastTrackStatus    string
	lastSessionStatus  string
	latestSeriesStatus string
	fastestLapOwner    string
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
	case SessionDataMsg:
		return sessionDataMsgHandler(m, msg)
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
		padding := lipgloss.PlaceHorizontal(
			m.width-4,
			lipgloss.Center,
			"",
			lipgloss.WithWhitespaceChars("."),
			lipgloss.WithWhitespaceForeground(subtle),
		)

		v = lipgloss.JoinVertical(
			lipgloss.Top,
			titleView(m),
			msgView(m, padding),
			subtitleView(m),
			tableView(m, padding),
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

type SessionDataMsg struct {
	SessionData f1livetiming.ChangeSessionData
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
		// Store fasted lap if we have one
		if newTiming.LastLapTime.OverallFastest {
			m.fastestLapOwner = key
		}
		// Merge timing data delta with existing data
		if oldTiming, ok := m.timingData[key]; ok {
			mergo.Merge(&oldTiming, newTiming, mergo.WithOverride)
		} else {
			m.timingData[key] = newTiming
		}
	}
	return m, updateTableCmd()
}

func sessionDataMsgHandler(m Model, msg SessionDataMsg) (Model, tea.Cmd) {
	latestTrackStatusKey := 0
	latestSessionStatusKey := 0
	for key, status := range msg.SessionData.StatusSeries {
		i, err := strconv.Atoi(key)
		if err != nil {
			m.logger.Debug("warning: SessionData.StatusSeries map key was not an integer - found", key)
			continue
		}
		if i > latestTrackStatusKey && status.TrackStatus != "" {
			m.logger.Debug("setting lastTrackStatus", status.TrackStatus)
			latestTrackStatusKey = i
			m.lastTrackStatus = status.TrackStatus
		}
		if i > latestSessionStatusKey && status.SessionStatus != "" {
			m.logger.Debug("setting lastSessionStatus", status.SessionStatus)
			latestSessionStatusKey = i
			m.lastSessionStatus = status.SessionStatus
		}
	}

	return m, updateTableCmd()
}

func updateTableMsgHandler(m Model, _ UpdateTableMsg) (Model, tea.Cmd) {
	rows := make([]table.Row, 0, 20)
	for driverNumber, data := range m.driverList {
		lastlap, pBest, oBest := getLastLap(m, driverNumber)

		lastlapStyle := styleYellow

		if lastlap == "-" {
			lastlapStyle = lipgloss.NewStyle()
		} else if oBest {
			lastlapStyle = stylePurple
		} else if pBest {
			lastlapStyle = styleGreen
		}

		rows = append(rows, table.NewRow(table.RowData{
			"position": data.Line,
			"driver":   getDriver(m, driverNumber),
			"interval": getInterval(m, driverNumber),
			"leader":   getLeaderGap(m, driverNumber),
			"lastlap":  table.NewStyledCell(lastlap, lastlapStyle),
		}))
	}

	m.table = newTable().WithRows(rows).SortByAsc("position")

	return m, nil
}

/* View Helper Functions
------------------------------------------------------------------------------------------------- */

func titleView(m Model) string {
	return h1Style.Width(m.width - 4).Render(m.sessionInfo.Meeting.Name)
}

func subtitleView(m Model) string {
	t := m.sessionInfo.Type

	if m.sessionInfo.Type == "Race" {
		t = fmt.Sprintf("%s: %d / %d Laps", t, m.lapCount.CurrentLap, m.lapCount.TotalLaps)
	}

	return h2Style.Width(m.width - 4).Render(t)
}

func msgView(m Model, p string) string {
	s := dialogBoxStyle
	msg := m.lastSessionStatus
	if m.lapCount.CurrentLap < m.lapCount.TotalLaps {
		msg = m.lastTrackStatus
	}

	switch msg {
	case "Ends":
		msg = "🏁 Session has ended 🏁"
		s.Border(lipgloss.BlockBorder())
	case "SCDeployed":
		msg = "Safety Car"
		s = dialogBoxStyle.BorderForeground(yellow)
	case "Yellow":
		msg = "🟨 Yellow Flag 🟨"
		s = dialogBoxStyle.BorderForeground(yellow)
	case "DoubleYellow":
		msg = "🟨 🟨 Double Yellow Flag 🟨 🟨"
		s = dialogBoxStyle.BorderForeground(yellow)
	case "AllClear":
		msg = "🟩 Green Flag 🟩"
		s = dialogBoxStyle.BorderForeground(green)
	case "Red":
		msg = "🟥 Red Flag 🟥"
		s = dialogBoxStyle.BorderForeground(red)
	}

	msgBox := lipgloss.PlaceHorizontal(
		m.width-4,
		lipgloss.Center,
		s.Width(m.width-10).Render(msg),
		lipgloss.WithWhitespaceChars(".."),
		lipgloss.WithWhitespaceForeground(subtle),
	)

	return lipgloss.JoinVertical(lipgloss.Top, p, p, msgBox, p, p)
}

func tableView(m Model, p string) string {
	t := lipgloss.PlaceHorizontal(
		m.width-4,
		lipgloss.Center,
		m.table.View(),
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(subtle),
	)

	return lipgloss.JoinVertical(lipgloss.Top, p, t, p)
}

func getDriver(m Model, driverNumber string) string {
	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(fmt.Sprintf("#%s", m.driverList[driverNumber].TeamColour))).
		PaddingLeft(1)
	name := nameStyle.Render(m.driverList[driverNumber].ShortName)

	if m.fastestLapOwner == driverNumber {
		return fmt.Sprintf("%s %s", name, stylePurple.Render("⏱"))
	}

	return name
}

func getInterval(m Model, driverNumber string) string {
	interval := "-"
	if m.timingData[driverNumber].Retired || m.timingData[driverNumber].Status == 4 {
		interval = "DNF"
	} else if m.timingData[driverNumber].IntervalToPositionAhead.Value != "" {
		interval = m.timingData[driverNumber].IntervalToPositionAhead.Value

		if m.timingData[driverNumber].IntervalToPositionAhead.Catching {
			interval = styleGreen.Render(interval)
		}
	}
	return interval
}

func getLeaderGap(m Model, driverNumber string) string {
	interval := "-"
	if m.timingData[driverNumber].Retired || m.timingData[driverNumber].Status == 4 {
		interval = "DNF"
	} else if m.timingData[driverNumber].GapToLeader != "" {
		interval = m.timingData[driverNumber].GapToLeader
	}
	return interval
}

func getLastLap(m Model, driverNumber string) (string, bool, bool) {
	lastlap := "-"
	pBest := m.timingData[driverNumber].LastLapTime.PersonalFastest
	oBest := m.timingData[driverNumber].LastLapTime.OverallFastest

	if !m.timingData[driverNumber].Retired &&
		m.timingData[driverNumber].Status != 4 &&
		m.timingData[driverNumber].LastLapTime.Value != "" {
		lastlap = m.timingData[driverNumber].LastLapTime.Value
	}

	return lastlap, pBest, oBest
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

func newTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn("position", "POS", 3),
		table.NewColumn("driver", "DRIVER", 7).WithStyle(lipgloss.NewStyle().Align(lipgloss.Left)),
		table.NewColumn("interval", "INT", 8),
		table.NewColumn("leader", "LEADER", 8),
		table.NewColumn("lastlap", "LAST", 10),
	}).
		WithRows([]table.Row{}).
		WithBaseStyle(lipgloss.NewStyle().AlignHorizontal(lipgloss.Center))
}
