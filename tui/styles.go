package tui

import "github.com/charmbracelet/lipgloss"

const (
	red = lipgloss.Color("#CF040E")
	// yellow       = lipgloss.Color("#A9B02B")
	yellow       = lipgloss.Color("#FAD105")
	green        = lipgloss.Color("#17C81D")
	purple       = lipgloss.Color("#DA0ED3")
	orange       = lipgloss.Color("#F77C14")
	wet          = lipgloss.Color("#1277EF")
	intermediate = lipgloss.Color("#2EA43F")
	hard         = lipgloss.Color("#D4DFE8")
	medium       = lipgloss.Color("#E4E344")
	soft         = lipgloss.Color("#FA5A55")
	fiaBlue      = lipgloss.Color("#0B203B")
	light        = lipgloss.Color("#D1D4DD")
	dark         = lipgloss.Color("#383838")
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
	msgCategoryStyle = lipgloss.NewStyle().
				Padding(1, 2).
				Bold(true).
				AlignVertical(lipgloss.Center)
	msgStyle = lipgloss.NewStyle().
			Background(light).
			Padding(1, 2).
			Foreground(dark).
			AlignVertical(lipgloss.Center)
)
