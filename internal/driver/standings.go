package driver

import (
	"fmt"
	"log"

	"github.com/bcdxn/f1cli/internal/f1scraper"
	"github.com/bcdxn/f1cli/internal/styles"
	"github.com/bcdxn/f1cli/internal/tealogger"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle           = lipgloss.NewStyle().Margin(1, 2)
	baseStyle          = lipgloss.NewStyle().Padding(0, 1)
	headerStyle        = baseStyle.Copy().Foreground(lipgloss.Color("252")).Bold(true)
	constructorsColors = map[string]lipgloss.Color{
		"Red Bull Racing": styles.RedbullColor,
		"Ferrari":         styles.FerrariColor,
		"McLaren":         styles.McLarenColor,
		"Mercedes":        styles.MercedesColor,
		"Aston Martin":    styles.AstonColor,
		"RB":              styles.RbColor,
		"Haas":            styles.HaasColor,
		"Williams":        styles.WilliamsColor,
		"Kick Sauber":     styles.SauberColor,
		"Alpine":          styles.AlpineColor,
	}
)

// Create a new application state object (called models in bubbletea)
func newTeaAppState(o StandingsOptions, sc f1scraper.F1ScraperClient, l tealogger.TeaLogger) teaAppState {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = styles.F1RedText

	return teaAppState{
		width:      0,
		height:     0,
		isLoading:  true,
		isQuitting: false,
		loadingMsg: styles.F1RedText.Render("Retrieving F1 Drivers Championship standings..."),
		spinner:    s,
		sc:         sc,
		l:          l,
	}
}

// The first hook called by bubbletea; we'll start a spinner
func (s teaAppState) Init() tea.Cmd {
	if s.isLoading {
		return s.spinner.Tick
	}
	return nil
}

// Update the application state appropriately based on the message
func (s teaAppState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		return windowSizeMsgHandler(s, msg)
	case tea.KeyMsg:
		return keyMsgHandler(s, msg)
	case StandingsMsg:
		return standingsMsgHandler(s, msg)
	default:
		return defaultHandler(s, msg)
	}
}

// Render the view
func (s teaAppState) View() string {
	str := ""
	if s.isLoading {
		str = docStyle.Render(fmt.Sprintf(
			"%s %s", s.spinner.View(),
			s.loadingMsg,
		))
	} else {
		str = docStyle.Render(
			styles.Title.Width(s.width).Render("F1 Drivers Standings") +
				"\n\n" + s.table.Render())
	}

	return str
}

type StandingsOptions struct {
	Debug bool
}

func RunProgram(o StandingsOptions) {
	l := tealogger.New(o.Debug)
	f := f1scraper.New(l)
	l.Debug("running driver standings program")
	p := tea.NewProgram(newTeaAppState(o, *f, l), tea.WithAltScreen())

	_, err := p.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
