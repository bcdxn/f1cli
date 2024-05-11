package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/bcdxn/f1cli/internal/tealogger"
)

type Schedule struct {
	Events []*RaceEvent
}

type RaceEvent struct {
	StartsAt        time.Time
	EndsAt          time.Time
	GmtOffset       string
	Location        string
	OfficialName    string
	Sessions        []*RaceEventSession
	Round           string
	EventDetailLink string
	Upcoming        bool
	IsHeroEvent     bool
}

type RaceEventSession struct {
	Name     string
	StartsAt time.Time
}

func (s Schedule) GetHeroEvent() *RaceEvent {
	for _, r := range s.Events {
		if r.IsHeroEvent {
			return r
		}
	}
	return nil
}

func (r RaceEvent) Title() string {
	return fmt.Sprintf("%s  %s", r.Location, r.formatConciseDates())
}

func (r RaceEvent) Description() string {
	return r.OfficialName
}

func (r RaceEvent) FilterValue() string {
	return r.Location
}

func (r RaceEvent) formatConciseDates() string {
	startMonth := r.StartsAt.Month()
	endMonth := r.EndsAt.Month()

	if startMonth == endMonth {
		return fmt.Sprintf("%s-%s", r.StartsAt.Format("Jan 2"), r.EndsAt.Format("2"))
	} else {
		return fmt.Sprintf("%s - %s", r.StartsAt.Format("Jan 2"), r.EndsAt.Format("Jan 2"))
	}
}

var (
	dayMap map[string]time.Weekday = map[string]time.Weekday{
		"mon": 0,
		"tue": 1,
		"wed": 2,
		"thu": 3,
		"fri": 4,
		"sat": 5,
		"sun": 6,
	}
)

func (r RaceEvent) GetSessionDate(startDay, startTime string) time.Time {
	startDay = strings.ToLower(startDay)
	currDay := r.StartsAt
	offset := 0

	for currDay.Weekday() != dayMap[startDay] {
		currDay = time.Now().AddDate(0, 0, 1)
		offset++
	}

	sessionDate := r.StartsAt.AddDate(0, 0, offset)
	year := sessionDate.Year()
	month := sessionDate.Month()
	day := sessionDate.Day()

	sessionDateTime, err := time.Parse("", fmt.Sprintf("%d-%d-%d %s", year, month, day, startTime))

	if err != nil {
		tealogger.LogErr(err)
	}

	return sessionDateTime
}
