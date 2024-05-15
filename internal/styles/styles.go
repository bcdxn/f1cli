package styles

import "github.com/charmbracelet/lipgloss"

const (
	F1Red = "#FF1801"
)

var (
	F1RedText     = lipgloss.NewStyle().Foreground(lipgloss.Color(F1Red))
	RedbullColor  = lipgloss.Color("#3671C6")
	FerrariColor  = lipgloss.Color("#E80020")
	McLarenColor  = lipgloss.Color("#FF8000")
	MercedesColor = lipgloss.Color("#27F4D2")
	AstonColor    = lipgloss.Color("#229971")
	RbColor       = lipgloss.Color("#6692FF")
	HaasColor     = lipgloss.Color("#B6BABD")
	WilliamsColor = lipgloss.Color("#64C4FF")
	SauberColor   = lipgloss.Color("#52E252")
	AlpineColor   = lipgloss.Color("#0093CC")
	Title         = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color(F1Red)).
			Foreground(lipgloss.Color("#FFFFFF")).
			PaddingLeft(1)
)
