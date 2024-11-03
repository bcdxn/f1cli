package tui

import "github.com/charmbracelet/lipgloss"

const (
	red    = lipgloss.Color("#CF040E")
	yellow = lipgloss.Color("#A9B02B")
	green  = lipgloss.Color("#17C81D")
	purple = lipgloss.Color("#DA0ED3")
	orange = lipgloss.Color("#F77C14")
)

var (
	styleSubtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	styleDoc    = lipgloss.NewStyle().Margin(1, 2)
	styleH1     = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Bold(true).
			PaddingBottom(1).
			Border(lipgloss.NormalBorder(), false, false, true, false)
	styleH2 = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false).
		Align(lipgloss.Center)
	stylePurple    = lipgloss.NewStyle().Foreground(lipgloss.Color(purple))
	styleGreen     = lipgloss.NewStyle().Foreground(lipgloss.Color(green))
	styleYellow    = lipgloss.NewStyle().Foreground(lipgloss.Color(yellow))
	styleDialogBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 1).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			Align(lipgloss.Center)
)
