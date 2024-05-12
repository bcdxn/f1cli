package schedule

import (
	"github.com/bcdxn/f1cli/internal/models"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
)

type teaAppState struct {
	width      int
	height     int
	isQuitting bool
	isLoading  bool
	loadingMsg string
	errMsg     string
	spinner    spinner.Model
	schedule   *models.Schedule
	list       list.Model
	hero       heroModel
}
