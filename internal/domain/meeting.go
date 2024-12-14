package domain

import (
	"time"
)

const (
	SessionTypeTest       SessionType = "Test"
	SessionTypePractice   SessionType = "Practice"
	SessionTypeQualifying SessionType = "Qualifying"
	SessionTypeRace       SessionType = "Race"
)

// NewMeeting returns a new instance of a meeting which represents data about a race weekend
// holistically as well as session-specific data as modeled per the domain with fields initialized
// to allow safe access (e.g. slices of appropriate length to prevent out of bounds indexing).
func NewMeeting() Meeting {
	return Meeting{
		Session: Session{
			GMTOffset:          "+0000",
			FastestSectorOwner: make([]string, 3),
		},
	}
}

// The types of sessions of within a race weekend, e.g.: Practice, Qualifying, Race, etc.
type SessionType string

// Meeting represents data about the race weekend event. This data applies to all of the sessions
// within a race weekend.
type Meeting struct {
	Name             string  // Name is the informal name of the race weekend event
	FullName         string  // FullName is the full official name of the event including primary sponsor
	Location         string  // Location is the locality in which the race weekend is taking place
	RoundNumber      int     // The sequence number of the race weekend event within the season
	CountryCode      string  // The 2-3 letter code indicating the country in which the event is taking place
	CountryName      string  // The full name of the country in which the event is taking place
	CircuitShortName string  // The informal name of the circuit at which the event is taking place
	Session          Session // A Race Weekend is composed of multiple sessions; only the active session is represented
}

// Session represents a specific session within a meeting, e.g.: Practice 1, Qualifying, Race
type Session struct {
	Type               SessionType
	Name               string    // The name of the session, e.g.: "Practice 1", "Race", etc.
	StartDate          time.Time // The start of the session
	EndDate            time.Time // The end time of the session - will be zerovalue until session has ended
	GMTOffset          string    // GMTOffset is the track-timezone delta with GMT/UTC
	FastestLapOwner    string    // FastestLapOwner is the number of the driver that has the fastest lap in the session
	FastestLapTime     string    // FastestLapTime is the time of the fastest lap of the session
	FastestSectorOwner []string  // The owner of the fastest time in each sector
	CurrentLap         int       // The current lead lap (only applicable for races)
	TotalLaps          int       // The total number of planned laps (only applicable for races)
	Part               int       // Part 0-based index, indicating the current part multi-part sessions, e.g.: Qualifying
}
