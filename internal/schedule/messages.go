package schedule

import "github.com/bcdxn/f1cli/internal/models"

type ErrorMsg error
type ScheduleMsg struct {
	schedule *models.Schedule
}
type EventDetailsMsg struct {
	sessions []*models.RaceEventSession
}
