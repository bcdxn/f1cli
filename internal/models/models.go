package models

import (
	"fmt"
	"time"
)

type RaceEvent struct {
	StartsAt     time.Time
	EndsAt       time.Time
	GmtOffset    string
	Location     string
	OfficialName string
}

func (r RaceEvent) Title() string {
	return fmt.Sprintf("%s %s", r.Location, r.getConciseDates())
}

func (r RaceEvent) Description() string {
	return r.OfficialName
}

func (r RaceEvent) FilterValue() string {
	return r.Location
}

func (r RaceEvent) getConciseDates() string {
	startMonth := r.StartsAt.Month()
	endMonth := r.EndsAt.Month()

	if startMonth == endMonth {
		return fmt.Sprintf("%s - %s", r.StartsAt.Format("Jan 2"), r.EndsAt.Format("2"))
	} else {
		return fmt.Sprintf("%s - %s", r.StartsAt.Format("Jan 2"), r.EndsAt.Format("Jan 2"))
	}
}
