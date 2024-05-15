package driver

import (
	"github.com/bcdxn/f1cli/internal/f1scraper"
	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/tealogger"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss/table"
)

type teaAppState struct {
	width      int
	height     int
	isQuitting bool
	isLoading  bool
	loadingMsg string
	errMsg     string
	spinner    spinner.Model
	standings  []*models.DriverStanding
	table      *table.Table
	l          tealogger.TeaLogger
	sc         f1scraper.F1ScraperClient
}
