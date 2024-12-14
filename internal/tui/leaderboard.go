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
	// case domain.SessionTypeQualifying:
	// 	return viewQualifyingTable(m)
	case domain.SessionTypeRace:
		return viewRaceTable(l)
	}

	return lipgloss.PlaceHorizontal(
		l.width,
		lipgloss.Center,
		t,
		lipgloss.WithWhitespaceChars("."),
		lipgloss.WithWhitespaceForeground(s.Color.Subtle),
	)
}

func viewRaceTable(l Leaderboard) string {
	baseStyle := lipgloss.NewStyle().Padding(0, 1, 1, 1)
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
		Headers("POS", "DRIVER", "INT", "LEADER", "LAST", "SECTORS", "STINT", "BEST").
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

		if m.Session.FastestLapTime == d.TimingData.LastLap.Time {
			v = lipgloss.NewStyle().Foreground(s.Color.Purple).Render(v)
		} else if d.TimingData.LastLap.IsPersonalBest {
			v = lipgloss.NewStyle().Foreground(s.Color.Green).Render(v)
		} else {
			v = lipgloss.NewStyle().Foreground(s.Color.Yellow).Render(v)
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

	v = d.TimingData.BestLapTime

	if d.TimingData.BestLapTime == m.Session.FastestLapTime {
		v = lipgloss.NewStyle().Foreground(lipgloss.Color(s.Color.Purple)).Render(v)
	}

	return v
}

func driverSectors(d domain.Driver, m domain.Meeting) string {
	if d.TimingData.IsKnockedOut || d.TimingData.IsRetired || len(d.TimingData.Sectors) < 1 {
		return s.Subtle.Render("-")
	}

	if d.TimingData.IsInPit {
		return disabledSectors()
	}

	if m.Session.Type == domain.SessionTypeQualifying && d.TimingData.IsPitOut {
		return s.Subtle.Render("OUT LAP ")
	}

	sectors := make([]string, 0, 3)
	// for i, sector := range d.Sectors {
	for i, sector := range d.TimingData.Sectors {
		sectorStyle := lipgloss.NewStyle()
		if !sector.IsActive {
			sectorStyle = sectorStyle.Foreground(s.Color.Subtle)
		} else if sector.IsOverallBest && m.Session.FastestSectorOwner[uint8(i)] == d.Number {
			sectorStyle = sectorStyle.Foreground(s.Color.Purple)
		} else if sector.IsPersonalBest {
			sectorStyle = sectorStyle.Foreground(s.Color.Green)
		} else {
			sectorStyle = sectorStyle.Foreground(s.Color.Yellow)
		}
		sectors = append(sectors, sectorStyle.Render("▃▃"))
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		sectors[0],
		" ",
		sectors[1],
		" ",
		sectors[2],
	)
}

func disabledSectors() string {
	sector := lipgloss.NewStyle().Foreground(s.Color.Subtle).Render("▃▃")
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		sector,
		" ",
		sector,
		" ",
		sector,
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

/* Tea Mesage Types
------------------------------------------------------------------------------------------------- */

type DriversMsg map[string]domain.Driver
type MeetingMsg domain.Meeting
type RaceCtrlMsgsMsg []domain.RaceCtrlMsg

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
	meeting  domain.Meeting
	drivers  map[string]domain.Driver
	isLoaded bool
	// metadata
	ctx    context.Context
	logger *slog.Logger
	// bubbles
	spinner spinner.Model
	// window size
	width  int
	height int
}
