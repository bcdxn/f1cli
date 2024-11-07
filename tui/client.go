package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
	timingAppData      map[string]f1livetiming.DriverTimingAppData
	bestLaps           map[string]string
	lapCount           f1livetiming.LapCount
	lastTrackStatus    string
	lastSessionStatus  string
	latestSeriesStatus string
	lastRaceControlMsg f1livetiming.RaceControlMessage
	fastestLapOwner    string
	fastestLap         string
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
	case RaceControlMsg:
		return raceCtrlMsgHandler(m, msg)
	case TimingAppDataMsg:
		return timingAppDataMsgHandler(m, msg)
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
			lipgloss.WithWhitespaceForeground(styleSubtle),
		)

		v = lipgloss.JoinVertical(
			lipgloss.Top,
			titleView(m),
			padding,
			padding,
			subtitleView(m),
			tableView(m, padding),
			msgView(m, padding),
		)
	}

	return styleDoc.Width(m.width).Render(v)
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
		bestLaps:           make(map[string]string, 20),
		timingAppData:      make(map[string]f1livetiming.DriverTimingAppData),
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

type RaceControlMsg struct {
	RaceControlMessage f1livetiming.RaceControlMessage
}

type TimingAppDataMsg struct {
	TimingAppData map[string]f1livetiming.DriverTimingAppData
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
	h, v := styleDoc.GetFrameSize()
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
	for key, new := range msg.TimingData {
		// Store fasted lap if we have one
		if new.LastLapTime.OverallFastest {
			m.fastestLapOwner = key
			m.fastestLap = new.LastLapTime.Value
		}
		if new.BestLapTime.Value != "" {
			m.bestLaps[key] = new.BestLapTime.Value
			if m.fastestLap == "" || new.BestLapTime.Value < m.fastestLap {
				m.fastestLap = new.BestLapTime.Value
				m.fastestLapOwner = key
			}
		}
		if new.LastLapTime.PersonalFastest {
			m.bestLaps[key] = new.LastLapTime.Value
		}
		// Merge timing data delta with existing data
		if oldTiming, ok := m.timingData[key]; ok {
			mergo.Merge(&oldTiming, new, mergo.WithOverride)
		} else {
			m.timingData[key] = new
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

func raceCtrlMsgHandler(m Model, msg RaceControlMsg) (Model, tea.Cmd) {
	m.lastRaceControlMsg = msg.RaceControlMessage
	return m, nil
}

func timingAppDataMsgHandler(m Model, msg TimingAppDataMsg) (Model, tea.Cmd) {
	for key, new := range msg.TimingAppData {
		if old, ok := m.timingAppData[key]; ok {
			mergo.Merge(&old, new, mergo.WithOverride)
		} else {
			m.timingAppData[key] = new
		}
	}
	return m, updateTableCmd()
}

func updateTableMsgHandler(m Model, _ UpdateTableMsg) (Model, tea.Cmd) {
	rows := make([]table.Row, 0, 20)
	for driverNumber, data := range m.driverList {
		interval, isDNF := getInterval(m, driverNumber)

		rows = append(rows, table.NewRow(table.RowData{
			"position": data.Line,
			"driver":   getDriver(m, driverNumber),
			"interval": interval,
			"leader":   getLeaderGap(m, driverNumber, isDNF),
			"tyre":     getTyre(m, driverNumber, isDNF),
			"lastlap":  getLastLap(m, driverNumber, isDNF),
			"sectors":  getSectors(m, driverNumber, isDNF),
			"bestlap":  getBaseLap(m, driverNumber),
		}))
	}

	m.table = newTable().WithRows(rows).SortByAsc("position")

	return m, nil
}

/* View Helper Functions
------------------------------------------------------------------------------------------------- */

func titleView(m Model) string {
	style := styleH1
	return style.Width(m.width - 4).Render(m.sessionInfo.Meeting.Name)
}

func subtitleView(m Model) string {
	t := m.sessionInfo.Name
	if m.sessionInfo.Type == "Race" {
		t = fmt.Sprintf("%d / %d Laps", m.lapCount.CurrentLap, m.lapCount.TotalLaps)
	}

	status := m.lastSessionStatus
	if m.lapCount.CurrentLap < m.lapCount.TotalLaps {
		status = m.lastTrackStatus
	}

	if m.lapCount.CurrentLap < m.lapCount.TotalLaps {
		status = m.lastTrackStatus
	}

	style := styleH2

	switch status {
	case "Ends":
		fallthrough
	case "Finished":
		fallthrough
	case "Finalised":
		t = ""
		status = fmt.Sprintf("%s has ended", m.sessionInfo.Name)
	case "SCDeployed":
		fallthrough
	case "SCEnding":
		status = "Safety Car"
		style = styleH2.BorderForeground(yellow)
	case "VSCDeployed":
	case "VSCEnding":
		status = "Virtual Safety Car"
		style = styleH2.BorderForeground(yellow)
	case "Yellow":
		status = "Yellow Flag"
		style = styleH2.BorderForeground(yellow)
	case "DoubleYellow":
		status = "Double Yellow Flag"
		style = styleH2.BorderForeground(yellow)
	case "AllClear":
		status = ""
		style = styleH2.BorderForeground(green)
	case "Red":
		status = "Red Flag"
		style = styleH2.BorderForeground(red)
	case "Aborted":
		status = "Aborted Start"
		style = styleH2.BorderForeground(red)
	}

	return style.Width(m.width - 4).Render(fmt.Sprintf("%s %s", t, status))
}

func msgView(m Model, p string) string {
	m.lastRaceControlMsg = f1livetiming.RaceControlMessage{
		UTC:      "2024-10-27T20:24:35",
		Lap:      13,
		Category: "Other",
		Message:  "FIA STEWARDS: TURN 4 INCIDENT INVOLVING CAR 1 (VER) UNDER INVESTIGATION - FORCING ANOTHER DRIVER OFF THE TRACK",
	}
	// m.lastRaceControlMsg = f1livetiming.RaceControlMessage{
	// 	UTC:      "2024-10-27T20:24:35",
	// 	Lap:      13,
	// 	Category: "Other",
	// 	Message:  "CAR 81 (PIA) TIME 1:22.312 DELETED - TRACK LIMITS AT TURN 2 LAP 38 14:59:43",
	// }
	// m.lastRaceControlMsg = f1livetiming.RaceControlMessage{
	// 	UTC:      "2024-10-27T20:24:35",
	// 	Lap:      13,
	// 	Category: "Flag",
	// 	Flag:     "DOUBLE YELLOW",
	// 	Message:  "DOUBLE YELLOW IN SECTOR asdfkja ;sdfkaj d;flakjsf ;alsdkfj a;sdlfj as;dlfkjasd ;flkjas d;flaksjd f;laksdjf a;dlskfj a;sdlkfj a;sdlfkj as;dlfj as;dlfjk a;sdfk jas;dlfkj as;dlfk jas;dfkljas;ldf kjas;dlkfj a;sdlfkj a;lsdfj a;sdlfkj as;dlfjk as;dlfkj a;sdlfj as;dlf jas;ldfj ",
	// }
	// m.lastRaceControlMsg = f1livetiming.RaceControlMessage{
	// 	UTC:      "2024-10-27T20:24:35",
	// 	Lap:      13,
	// 	Category: "Flag",
	// 	Flag:     "CLEAR",
	// 	Message:  "TRACK IS ALL CLEAR",
	// }
	// m.lastRaceControlMsg = f1livetiming.RaceControlMessage{
	// 	UTC:      "2024-10-27T20:04:31",
	// 	Lap:      1,
	// 	Category: "SafetyCar",
	// 	Status:   "DEPLOYED",
	// 	Mode:     "SAFETY CAR",
	// 	Message:  "SAFETY CAR DEPLOYED",
	// }
	// s := styleDialogBox
	msg := m.lastRaceControlMsg.Message
	if msg == "" {
		return lipgloss.JoinVertical(lipgloss.Top, p, p, p, p, p, p, p, p, p, p)
	}

	// t, err := time.Parse("2006-01-02T15:04:05", m.lastRaceControlMsg.UTC)
	// if err != nil {
	// 	m.logger.Debug("unable to parse race control message time:", m.lastRaceControlMsg)
	// 	return lipgloss.JoinVertical(lipgloss.Top, p, p, p, p, p, p, p, p, p, p)
	// }
	// if time.Since(t).Seconds() > 15 {
	// 	return lipgloss.JoinVertical(lipgloss.Top, p, p, p, p, p, p, p, p, p, p)
	// }

	cStyle := msgCategoryStyle
	category := ""
	mStyle := msgStyle.Width(60)
	cStyle = cStyle.Height(msgStyle.GetHeight())

	switch m.lastRaceControlMsg.Category {
	case "Flag":
		switch m.lastRaceControlMsg.Flag {
		case "YELLOW":
			fallthrough
		case "DOUBLE YELLOW":
			category = strings.Join(strings.Split(m.lastRaceControlMsg.Flag, " "), "\n")
			cStyle = cStyle.Background(yellow).Foreground(dark)
		case "CLEAR":
			category = "ALL\nCLEAR"
			cStyle = cStyle.Background(green).Foreground(light)
		case "RED":
			category = strings.Join(strings.Split(m.lastRaceControlMsg.Flag, " "), "\n")
			cStyle = cStyle.Background(red).Foreground(dark)
		case "BLUE":
			return lipgloss.JoinVertical(lipgloss.Top, p, p, p, p, p, p, p, p, p, p)
		case "CHEQUERED":
		}
	case "SafetyCar":
		category = strings.Join(strings.Split(m.lastRaceControlMsg.Mode, " "), "\n")
		cStyle = cStyle.Background(yellow).Foreground(dark)
	case "Other":
		cStyle = cStyle.Background(fiaBlue).Foreground(light)
		category = "RACE\nCONTROL"
		fia := regexp.MustCompile("FIA STEWARDS: ")
		investigation := regexp.MustCompile("UNDER INVESTIGATION")
		penalty := regexp.MustCompile("PENALTY FOR")

		if fia.MatchString(msg) || investigation.MatchString(msg) || penalty.MatchString(msg) {
			category = "FIA"
		}

		msg = fia.ReplaceAllString(msg, "")
	}

	// .Background(lipgloss.Color("#0b203b")).Foreground(lipgloss.Color("#d1d4dd"))

	// switch m.lastRaceControlMsg.Category {
	// case "Flag":
	// 	switch m.lastRaceControlMsg.Flag {
	// 	case "DOUBLE YELLOW":
	// 		s.Foreground(yellow).BorderForeground(yellow)
	// 	case "YELLOW":
	// 		s.Foreground(yellow).BorderForeground(yellow)
	// 	case "CLEAR":
	// 		s.Foreground(green).BorderForeground(green)
	// 	case "RED":
	// 		s.Foreground(red).BorderForeground(red)
	// 	case "BLUE":
	// 		return lipgloss.JoinVertical(lipgloss.Top, p, p, p, p, p, p, p, p, p, p)
	// 	case "CHEQUERED": // nothing special to do here
	// 	}
	// }

	// if investigation.MatchString(msg) {
	// s = s.Foreground(orange).BorderForeground(orange)
	renderedCat := cStyle.Render(category)
	renderedMsg := mStyle.Render(msg)

	if lipgloss.Height(renderedCat) > lipgloss.Height(renderedMsg) {
		renderedMsg = mStyle.Height(lipgloss.Height(renderedCat)).Render(msg)
	} else {
		renderedCat = cStyle.Height(lipgloss.Height(renderedMsg)).Render(category)
	}

	test := lipgloss.JoinHorizontal(
		lipgloss.Center,
		renderedCat,
		renderedMsg,
	)
	// 	fia.ReplaceAllString(msg, "")
	// } else if penalty.MatchString(msg) {
	// 	s.Foreground(red).BorderForeground(red)
	// }

	msgBox := lipgloss.PlaceHorizontal(
		m.width-4,
		lipgloss.Center,
		// s.Width(m.width-10).Render(msg),
		test,
		lipgloss.WithWhitespaceChars(".."),
		lipgloss.WithWhitespaceForeground(styleSubtle),
	)

	return lipgloss.JoinVertical(lipgloss.Top, p, p, msgBox, p, p)
}

func tableView(m Model, p string) string {
	t := lipgloss.PlaceHorizontal(
		m.width-4,
		lipgloss.Center,
		m.table.View(),
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(styleSubtle),
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

func getInterval(m Model, driverNumber string) (string, bool) {
	isDNF := false
	interval := "-"
	if m.timingData[driverNumber].Retired || m.timingData[driverNumber].Status == 4 {
		interval = "DNF"
		isDNF = true
	} else if m.timingData[driverNumber].IntervalToPositionAhead.Value != "" {
		interval = m.timingData[driverNumber].IntervalToPositionAhead.Value

		if m.timingData[driverNumber].IntervalToPositionAhead.Catching {
			interval = styleGreen.Render(interval)
		}
	} else if m.timingData[driverNumber].TimeDiffToPositionAhead != "" {
		interval = m.timingData[driverNumber].TimeDiffToPositionAhead
	}
	return interval, isDNF
}

func getLeaderGap(m Model, driverNumber string, isDNF bool) string {
	interval := "-"
	if isDNF {
		interval = "-"
	} else if m.timingData[driverNumber].GapToLeader != "" {
		interval = m.timingData[driverNumber].GapToLeader
	} else if m.timingData[driverNumber].TimeDiffToFastest != "" {
		interval = m.timingData[driverNumber].TimeDiffToFastest
	}
	return interval
}

func getLastLap(m Model, driverNumber string, isDNF bool) string {
	lastlap := "-"

	if isDNF {
		return lastlap
	}

	if m.timingData[driverNumber].LastLapTime.Value != "" {
		lastlap = m.timingData[driverNumber].LastLapTime.Value
	}

	lastlapStyle := styleYellow
	if m.timingData[driverNumber].LastLapTime.OverallFastest {
		lastlapStyle = stylePurple
	} else if m.timingData[driverNumber].LastLapTime.PersonalFastest {
		lastlapStyle = styleGreen
	}

	return lastlapStyle.Render(lastlap)
}

func getTyre(m Model, driverNumber string, isDNF bool) string {
	if isDNF {
		return "-"
	}

	l := len(m.timingAppData[driverNumber].Stints)
	s := m.timingAppData[driverNumber].Stints[strconv.Itoa(l-1)]
	c := s.Compound
	style := lipgloss.NewStyle()

	switch c {
	case "SOFT":
		c = style.Foreground(soft).Render("S")
	case "MEDIUM":
		c = style.Foreground(medium).Render("M")
	case "HARD":
		c = style.Foreground(hard).Render("H")
	case "INTERMEDIATE":
		c = style.Foreground(intermediate).Render("I")
	case "WET":
		c = style.Foreground(wet).Render("W")
	default:
		c = style.Render("-")
	}

	return fmt.Sprintf("%s %d laps", c, s.TotalLaps)
}

func getSectors(m Model, driverNumber string, isDNF bool) string {
	if isDNF {
		return "-"
	}

	var s1 f1livetiming.SectorTiming
	var s2 f1livetiming.SectorTiming
	var s3 f1livetiming.SectorTiming

	if len(m.timingData[driverNumber].Sectors) > 0 {
		s1 = m.timingData[driverNumber].Sectors[0]
	}
	if len(m.timingData[driverNumber].Sectors) > 1 {
		s2 = m.timingData[driverNumber].Sectors[1]
	}
	if len(m.timingData[driverNumber].Sectors) > 2 {
		s3 = m.timingData[driverNumber].Sectors[2]
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		getSector(s1),
		" ",
		getSector(s2),
		" ",
		getSector(s3),
	)
}

func getSector(s f1livetiming.SectorTiming) string {
	style := lipgloss.NewStyle()
	if s.Value == "" {
		return style.Foreground(styleSubtle).Render("▃▃")
	}
	if s.OverallFastest {
		return style.Foreground(purple).Render("▃▃")
	}
	if s.PersonalFastest {
		return style.Foreground(green).Render("▃▃")
	}
	return style.Foreground(yellow).Render("▃▃")
}

func getBaseLap(m Model, driverNumber string) string {
	if m.bestLaps[driverNumber] == "" {
		return "-"
	}
	return m.bestLaps[driverNumber]
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

func newTable() table.Model {
	return table.New([]table.Column{
		table.NewColumn("position", "POS", 3),
		table.NewColumn("driver", "DRIVER", 7).WithStyle(lipgloss.NewStyle().Align(lipgloss.Left)),
		table.NewColumn("interval", "INTERVAL", 8),
		table.NewColumn("leader", "LEADER", 8),
		table.NewColumn("tyre", "TYRE", 9),
		table.NewColumn("lastlap", "LAST", 10),
		table.NewColumn("sectors", "SECTORS", 10),
		table.NewColumn("bestlap", "BEST", 10),
	}).
		WithRows([]table.Row{}).
		WithBaseStyle(lipgloss.NewStyle().AlignHorizontal(lipgloss.Center))
}
