package domain

const (
	TireCompoundSoft         TireCompound = "SOFT"
	TireCompoundMedium       TireCompound = "MEDIUM"
	TireCompoundHard         TireCompound = "HARD"
	TireCompoundIntermediate TireCompound = "INTERMEDIATE"
	TireCompoundFullWet      TireCompound = "WET"
	TireCompoundTest         TireCompound = "TEST"
	TireCompoundUnknown      TireCompound = "UNKNOWN"
)

// NewDriver returns a new instance of a driver as modeled per the domain with fields initialized
// to allow safe access (e.g. slices of appropriate length to prevent out of bounds indexing).
func NewDriver(number string) Driver {
	return Driver{
		Number: number,
		TimingData: DriverTimingData{
			ShowPosition: true,
			Sectors:      make([]Sector, 3),
			BestLapTimes: make([]string, 3),
			TireCompound: TireCompoundUnknown,
		},
	}
}

// TireCompound represents one of the official tire compound types used in a race weekend.
type TireCompound string

// Driver domain model represent intrinsic data about a driver as well as updates to live-timing
// data like grid position, gaps, etc.
type Driver struct {
	// Intrinsic Data
	Number     string // Number is the unique driver racing number present on their car
	ShortName  string // Shortname is the name abbreviation used on the television broadcast
	Name       string // Name is the full name of the driver
	TeamName   string // TeamName is the short name of the team that the driver races for
	TeamColor  string // TeamColor is the primary color of the team that the driver races for
	TimingData DriverTimingData
}

// Driver domain model represents intrinsic data about a driver as well as updates to live-timing
// data like grid position, gaps, etc.
type DriverTimingData struct {
	// Timing data
	Position    int      // Position is the driver's position on the timing board
	IntervalGap string   // IntervalGap is the time delta between the driver and the driver ahead
	LeaderGap   string   // LeaderGap is the delta between the driver and the lead driver
	LastLap     struct { // Data about the last completed lap
		Time           string // Time is The lap time of the last lap
		IsPersonalBest bool   // PersonalBest indicates if the last lap is a personal best for the driver
	}
	BestLapTime string // BestLapTime is the time of the best lap
	// Stint Data
	TireCompound TireCompound // The current tire compound that the driver is using
	TireLapCount int          // The current lap count that the driver is on
	IsInPit      bool         // InPit indicates if the driver is in the pit
	ShowPosition bool         // The driver is out of the session due to crash, mechanical failure, etc.
	IsPitOut     bool         // PitOut indicates if the driver is on an outlap
	// Sector times
	Sectors []Sector
	// Race-specific data
	NumberOfLaps int
	IsRetired    bool // The driver is out of the session due to crash, mechanical failure, etc.
	// Qualifying-specific data
	BestLapTimes []string // Best times in each session part (applicable for Qualifying sessions only, e.g.: Q1, Q2, Q3)
	IsKnockedOut bool     // The driver did not qualify for the current session (only applicable during qualifiying session)
	Cutoff       bool     // The driver is in the cutoff zone (only applicable during qualifiying session)
}

// Sector represents timing data about individual sectors around the lap.
type Sector struct {
	Time           string
	IsPersonalBest bool
	IsOverallBest  bool
	IsActive       bool
}
