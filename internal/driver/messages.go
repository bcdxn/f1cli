package driver

import "github.com/bcdxn/f1cli/internal/models"

type ErrorMsg error
type StandingsMsg struct {
	standings []*models.DriverStanding
}
