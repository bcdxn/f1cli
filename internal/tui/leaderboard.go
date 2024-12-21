package tui

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/bcdxn/f1cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/reflow/wordwrap"
)

var (
	s = styles.Default()
)

func NewLeaderboard(opts ...TUIOption) *tea.Program {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot

	l := Leaderboard{
		drivers: make(map[string]domain.Driver),
		logger:  slog.Default(),
		ctx:     context.Background(),
		spinner: sp,
	}
	// apply given options
	for _, opt := range opts {
		opt(&l)
	}
	// return new Bubbletea program
	return tea.NewProgram(l, tea.WithContext(l.ctx), tea.WithAltScreen())
}

type TUIOption = func(c *Leaderboard)

// WithLogger configures the logger to use within the TUI program
func WithLogger(l *slog.Logger) TUIOption {
	return func(b *Leaderboard) { b.logger = l }
}

// WithLogger configures the context to use within the TUI program
func WithContext(ctx context.Context) TUIOption {
	return func(b *Leaderboard) { b.ctx = ctx }
}

/* Bubbletea Interface Implementation
------------------------------------------------------------------------------------------------- */

func (l Leaderboard) Init() tea.Cmd {
	return l.spinner.Tick
}

func (l Leaderboard) View() string {
	var v string

	if !l.isLoaded {
		v = l.spinner.View() + " loading..."
	} else {
		v = lipgloss.JoinVertical(
			lipgloss.Center,
			viewHeader(l),
			viewPadding(l),
			viewTable(l),
			viewPadding(l),
			viewRaceCtrlMsg(l),
			viewPadding(l),
		)
	}

	return s.Doc.Width(l.width).Render(v)
}

func (l Leaderboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return handleKeyMsg(l, msg)
	case tea.WindowSizeMsg:
		return handleWindowSizeMsg(l, msg)
	case MeetingMsg:
		l.meeting = domain.Meeting(msg)
		l.isLoaded = true
	case DriversMsg:
		l.drivers = map[string]domain.Driver(msg)
		l.isLoaded = true
	case RaceCtrlMsg:
		l.raceCtrlMsg = domain.RaceCtrlMsg(msg)
	default:
		if !l.isLoaded {
			l.spinner, cmd = l.spinner.Update(msg)
		}
	}

	return l, cmd
}

/* View Helpers
------------------------------------------------------------------------------------------------- */

// getPadding returns the padding view component
func viewPadding(m Leaderboard) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Center,
		"",
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(s.Color.Subtle),
	)
}

// viewHeader returns the header view component
func viewHeader(l Leaderboard) string {
	titleBarStyle := s.TitleBar
	subtitleBarStyle := s.SubtitleBar

	subtitleContent := l.meeting.Name
	if l.meeting.Session.Type == domain.SessionTypeRace {
		subtitleContent = fmt.Sprintf("Race: %d / %d Laps", l.meeting.Session.CurrentLap, l.meeting.Session.TotalLaps)
	} else if l.meeting.Session.Type == domain.SessionTypeQualifying {
		subtitleContent = fmt.Sprintf("Qualifying %d", l.meeting.Session.Part)
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleBarStyle.Width(l.width).Render(l.meeting.FullName),
		subtitleBarStyle.Width(l.width).Render(subtitleContent),
	)
}

func viewTable(l Leaderboard) string {
	t := ""
	switch l.meeting.Session.Type {
	case domain.SessionTypeQualifying:
		t = viewQualifyingTable(l)
	case domain.SessionTypeRace:
		t = viewRaceTable(l)
	}

	return lipgloss.PlaceHorizontal(
		l.width,
		lipgloss.Center,
		t,
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(s.Color.Subtle),
	)
}

func viewQualifyingTable(l Leaderboard) string {
	baseStyle := s.TableRow
	drivers := sortDrivers(l.drivers)
	rows := make([][]string, 0, len(drivers))

	for _, d := range drivers {
		rows = append(rows, []string{
			driverPosition(d),
			driverName(d, l.meeting),
			driverIntervalGap(d),
			driverLeaderGap(d),
			driverSectors(d, l.meeting),
			driverBestLapInPart(d, 0),
			driverBestLapInPart(d, 1),
			driverBestLapInPart(d, 2),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		// BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := baseStyle

			if row == len(rows)-1 {
				style = style.Padding(0, 1)
			}

			if col == 0 {
				style = style.Align(lipgloss.Right)
			}

			return style
		}).
		Headers("POS", "DRIVER", "INT", "LEADER", "MINI SECTORS", "Q1 BEST", "Q2 BEST", "Q3 BEST").
		Rows(rows...)

	return t.Render()
}

func viewRaceTable(l Leaderboard) string {
	baseStyle := s.TableRow
	drivers := sortDrivers(l.drivers)
	rows := make([][]string, 0, len(drivers))

	for _, d := range drivers {
		rows = append(rows, []string{
			driverPosition(d),
			driverName(d, l.meeting),
			driverIntervalGap(d),
			driverLeaderGap(d),
			driverLastLap(d, l.meeting),
			driverSectors(d, l.meeting),
			driverStint(d),
			driverBestLap(d, l.meeting),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := baseStyle

			if row == len(rows)-1 {
				style = style.Padding(0, 1)
			}
			if col == 0 {
				style = style.Align(lipgloss.Right)
			}

			return style
		}).
		Headers("POS", "DRIVER", "INT", "LEADER", "LAST", "MINI SECTORS", "TIRE", "BEST").
		Rows(rows...)

	return t.Render()
}

func driverPosition(d domain.Driver) string {
	v := "-"
	if pos := d.TimingData.Position; pos != 0 {
		v = strconv.Itoa(pos)
	}
	if d.TimingData.IsRetired {
		v = "DNF"
	}
	if d.TimingData.IsKnockedOut || d.TimingData.IsRetired || !d.TimingData.ShowPosition {
		v = lipgloss.NewStyle().Foreground(s.Color.Subtle).Render(v)
	}
	return v
}

// driverName returns the driver name formatted with the team color and fastsest lap indicator when
// appropriate formatted for the timing table
func driverName(d domain.Driver, m domain.Meeting) string {
	c := lipgloss.Color(d.TeamColor)
	n := lipgloss.NewStyle().Foreground(c).Render("▍")

	if d.TimingData.IsKnockedOut || d.TimingData.IsRetired {
		n += lipgloss.NewStyle().Foreground(s.Color.Subtle).Render(d.ShortName) + " "
	} else {
		n += d.ShortName + " "
	}

	if m.Session.Type == domain.SessionTypeRace && d.Number == m.Session.FastestLapOwner {
		n += s.Purple.Render("⏱")
		return n
	} else if !d.TimingData.IsRetired && d.TimingData.IsInPit {
		n += lipgloss.NewStyle().Foreground(s.Color.Subtle).Render("P")
	} else if m.Session.Type == domain.SessionTypeQualifying && !d.TimingData.IsKnockedOut {
		n += driverTireCompound(d)
	}
	return n
}

var (
	leaderRe = regexp.MustCompile(`LAP`)
)

// driverIntervalGap returns the driver interval to the car ahead formatted for the timing table.
func driverIntervalGap(d domain.Driver) string {
	if d.TimingData.IntervalGap == "" || leaderRe.MatchString(d.TimingData.IntervalGap) {
		return "-"
	}
	if d.TimingData.IsRetired || d.TimingData.IsKnockedOut {
		return s.Subtle.Render("-")
	}
	return d.TimingData.IntervalGap
}

// driverLeaderGap returns the driver interval to the leader car formatted for the timing table.
func driverLeaderGap(d domain.Driver) string {
	if d.TimingData.LeaderGap == "" || d.TimingData.IsRetired || d.TimingData.IsKnockedOut || leaderRe.MatchString(d.TimingData.LeaderGap) {
		return "-"
	}
	return d.TimingData.LeaderGap
}

// driverLeaderGap returns the driver interval to the leader car formatted for the timing table.
func driverTireCompound(d domain.Driver) string {
	if d.TimingData.TireCompound == "" || d.TimingData.IsRetired {
		return "-"
	}
	t := d.TimingData.TireCompound[:1]
	tireStyle := lipgloss.NewStyle()
	switch d.TimingData.TireCompound {
	case domain.TireCompoundSoft:
		tireStyle = tireStyle.Foreground(s.Color.SoftTire)
	case domain.TireCompoundMedium:
		tireStyle = tireStyle.Foreground(s.Color.MediumTire)
	case domain.TireCompoundIntermediate:
		tireStyle = tireStyle.Foreground(s.Color.IntermediateTire)
	case domain.TireCompoundFullWet:
		tireStyle = tireStyle.Foreground(s.Color.WetTire)
	case domain.TireCompoundUnknown:
		t = "X"
	}

	return tireStyle.Render(string(t))
}

func driverStint(d domain.Driver) string {
	if d.TimingData.IsRetired {
		return s.Subtle.Render("-")
	}

	return fmt.Sprintf("%s %d Laps", driverTireCompound(d), d.TimingData.TireLapCount)
}

func driverLastLap(d domain.Driver, m domain.Meeting) string {
	v := "-"

	if d.TimingData.LastLap.Time != "" {
		v = d.TimingData.LastLap.Time

		if d.TimingData.IsRetired {
			v = s.Subtle.Render(v)
		} else if d.Number == m.Session.FastestLapOwner && d.TimingData.LastLap.Time == d.TimingData.BestLapTime {
			v = s.Purple.Render(v)
		} else if d.TimingData.LastLap.IsPersonalBest {
			v = s.Green.Render(v)
		} else {
			v = s.Yellow.Render(v)
		}
	}

	return v
}

func driverBestLap(d domain.Driver, m domain.Meeting) string {
	v := d.TimingData.BestLapTime

	if d.TimingData.BestLapTime == "" {
		v = "-"
	}

	if d.TimingData.IsKnockedOut || d.TimingData.IsRetired {
		return s.Subtle.Render(v)
	}

	if d.Number == m.Session.FastestLapOwner {
		v = lipgloss.NewStyle().Foreground(lipgloss.Color(s.Color.Purple)).Render(v)
	}

	return v
}

func driverBestLapInPart(d domain.Driver, part int) string {
	v := d.TimingData.BestLapTimes[part]

	if v == "" {
		v = "-"
	}

	if d.TimingData.IsKnockedOut || d.TimingData.IsRetired {
		return s.Subtle.Render(v)
	}

	return v
}

func driverSectors(d domain.Driver, m domain.Meeting) string {
	if d.TimingData.IsKnockedOut || d.TimingData.IsRetired || len(d.TimingData.Sectors) < 1 {
		return s.Subtle.Render("-")
	}

	if m.Session.Type == domain.SessionTypeQualifying && d.TimingData.IsPitOut {
		return s.Subtle.Render("OUT LAP ")
	}

	segments := make([]string, 0)

	// iterate through the sectors (there's always 3)
	for i := 0; i < 3; i++ {
		// iterate through the segments in order (there's a variable number)
		secNum := strconv.Itoa(i)
		segKeys := make([]string, 0, len(d.TimingData.Sectors[secNum].Segments))
		for k := range d.TimingData.Sectors[secNum].Segments {
			segKeys = append(segKeys, k)
		}
		sort.Strings(segKeys)
		for _, segKey := range segKeys {
			switch d.TimingData.Sectors[secNum].Segments[segKey].Status {
			case domain.SectorStatusNotPersonalBest:
				segments = append(segments, s.Yellow.Render("▍"))
			case domain.SectorStatusPersonalBest:
				segments = append(segments, s.Green.Render("▍"))
			case domain.SectorStatusOverallBest:
				segments = append(segments, s.Purple.Render("▍"))
			default:
				segments = append(segments, s.Subtle.Render("▍"))
			}
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		segments...,
	)
}

// sortDrivers returns a sorted of list of drivers, sorted by their leaderboard position in the
// session used by the timing table.
func sortDrivers(driverMap map[string]domain.Driver) []domain.Driver {
	drivers := make([]domain.Driver, 0, len(driverMap))
	for _, driver := range driverMap {
		drivers = append(drivers, driver)
	}

	sort.Slice(drivers, func(i, j int) bool {
		p1 := drivers[i].TimingData.Position
		p2 := drivers[j].TimingData.Position
		if drivers[i].TimingData.IsRetired {
			// DNF drivers should appear at the bottom of the timing board and be ordered by number of
			// laps completed
			p1 = 100 - drivers[i].TimingData.NumberOfLaps
		}
		if drivers[j].TimingData.IsRetired {
			p2 = 100 - drivers[j].TimingData.NumberOfLaps
		}
		// In the case that two drivers DNF on the same lap or otherwise are reported to have the same
		// position simply rank them by their driver number
		if p1 == p2 {
			num1, _ := strconv.Atoi(drivers[i].Number)
			num2, _ := strconv.Atoi(drivers[j].Number)
			return p1-num1 < p2-num2
		}

		return p1 < p2
	})

	return drivers
}

func viewRaceCtrlMsg(l Leaderboard) string {
	title := l.raceCtrlMsg.Title
	body := l.raceCtrlMsg.Body
	var titleStyle lipgloss.Style
	var bodyStyle lipgloss.Style
	switch l.raceCtrlMsg.Category {
	case domain.RaceCtrlMsgCategoryFIA:
		titleStyle = s.ToastMsgTitle.Background(s.Color.FiaBlue).Foreground(s.Color.Light)
		bodyStyle = s.ToastMsgBody.Background(s.Color.Light).Foreground(s.Color.FiaBlue)
	case domain.RaceCtrlMsgCategoryTrackStatus:
		bodyStyle = s.ToastMsgBody.Background(s.Color.Light).Foreground(s.Color.Dark)
		switch l.raceCtrlMsg.Title {
		case domain.RaceCtrlMsgTitleFlagBlue:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Blue).Foreground(s.Color.Dark)
		case domain.RaceCtrlMsgTitleFlagYellow:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Yellow).Foreground(s.Color.Dark)
		case domain.RaceCtrlMsgTitleFlagDoubleYellow:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Yellow).Foreground(s.Color.Dark)
		case domain.RaceCtrlMsgTitleVSC:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Yellow).Foreground(s.Color.Dark)
		case domain.RaceCtrlMsgTitleSC:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Yellow).Foreground(s.Color.Dark)
		case domain.RaceCtrlMsgTitleFlagBW:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Dark).Foreground(s.Color.Light)
		case domain.RaceCtrlMsgTitleFlagRed:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Red).Foreground(s.Color.Light)
		case domain.RaceCtrlMsgTitleFlagGreen:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Green).Foreground(s.Color.Dark)
		default:
			titleStyle = s.ToastMsgTitle.Background(s.Color.Dark)
		}
	}

	renderedTitle := titleStyle.Render(title)
	renderedBody := bodyStyle.Render(wordwrap.String(body, bodyStyle.GetMaxWidth()-(bodyStyle.GetPaddingLeft()+bodyStyle.GetPaddingRight())))

	if lipgloss.Height(renderedTitle) > lipgloss.Height(renderedBody) {
		renderedBody = bodyStyle.Height(lipgloss.Height(renderedTitle)).Render(body)
	} else {
		renderedTitle = titleStyle.Height(lipgloss.Height(renderedBody)).Render(title)
	}

	return lipgloss.PlaceHorizontal(
		l.width,
		lipgloss.Center,
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			renderedTitle,
			renderedBody,
		),
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(s.Color.Subtle),
	)
}

/* Tea Mesage Types
------------------------------------------------------------------------------------------------- */

type DriversMsg map[string]domain.Driver
type MeetingMsg domain.Meeting
type RaceCtrlMsg domain.RaceCtrlMsg

/* Tea Mesage handlers
------------------------------------------------------------------------------------------------- */

// handleKeyMsg is a tea.Msg handler that handles key press messages including ctrl+c and q to quit
// the TUI application.
func handleKeyMsg(m Leaderboard, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.logger.Debug("received quit tea message")
		return m, tea.Quit
	}
	return m, nil
}

// handleWindowSizeMsg is a tea.Msg handler that handles window resize events and stores the current
// window size of the terminal in the tea model.
func handleWindowSizeMsg(l Leaderboard, msg tea.WindowSizeMsg) (Leaderboard, tea.Cmd) {
	h, v := s.Doc.GetFrameSize()
	l.width = msg.Width - h
	l.height = msg.Height - v
	return l, nil
}

/* Type Definitions
------------------------------------------------------------------------------------------------- */

type Leaderboard struct {
	// leaderboard state
	meeting     domain.Meeting
	drivers     map[string]domain.Driver
	raceCtrlMsg domain.RaceCtrlMsg
	isLoaded    bool
	// metadata
	ctx    context.Context
	logger *slog.Logger
	// bubbles
	spinner spinner.Model
	// window size
	width  int
	height int
}
