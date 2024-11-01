package tui

import "github.com/charmbracelet/lipgloss"

const (
	red    = lipgloss.Color("#CF040E")
	yellow = lipgloss.Color("#A9B02B")
	green  = lipgloss.Color("#17C81D")
	purple = lipgloss.Color("#DA0ED3")
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)
	h1Style  = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false)
	h2Style        = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false).Align(lipgloss.Center)
	stylePurple    = lipgloss.NewStyle().Foreground(lipgloss.Color(purple))
	styleGreen     = lipgloss.NewStyle().Foreground(lipgloss.Color(green))
	styleYellow    = lipgloss.NewStyle().Foreground(lipgloss.Color(yellow))
	subtle         = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 1).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			Align(lipgloss.Center)
)
