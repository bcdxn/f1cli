package f1livetiming

import (
	"encoding/json"
	"strconv"
	"time"
)

const (
	statusCrashDamageRetiredOnTrack = 68
	statusCrashDamageRetiredInPit   = 92
	// statusGainedPlaces              = 4160
	// statusLostPlaces                = 8256
	// statusWithinVSCDelta            = 64
	// statusInPit                     = 80

	yellowSegment = 2048
	greenSegment  = 2049
	purpleSegment = 2051
	pitSegment    = 2064
)

// f1Message represents a websocket message from the F1 Live Timing API. It comes in two primary
// varieties: Change messages and Reference messages. There is a single Reference message sent at
// the beginning of the websocket connection, followed by updates via Change maessages.
type f1Message struct {
	Changes   json.RawMessage `json:"M"`
	Reference json.RawMessage `json:"R"`
}

// f1ChangeMessage represents a 'change' message sent on the websocket connection from the server.
// It is a delta between the reference data and any other preceeding change messages.
type f1ChangeMessage struct {
	Arguments []json.RawMessage `json:"A"`
}

// f1ReferenceMessance represents the initial state of a session for all of the requested data from
// the F1 Live Timing API. This includes intrinsic data about the session as well as driver, timing
// and status data. The reference message should be used to create an initial state; all other
// messages are 'Change' data messages that alter the state managed by the API consumer.
type f1ReferenceMessage struct {
	Heartbeat     json.RawMessage `json:"Heartbeat"`           // Heartbeat is the most recent heartbeat emitted
	TimingAppData json.RawMessage `json:"TimingAppData"`       // TimingAppData contains per-driver stint information
	DriverList    json.RawMessage `json:"DriverList"`          // DriverList contains per-driver intrinsic data
	RaceCtrlMsgs  json.RawMessage `json:"RaceControlMessages"` // RaceCtrlMsgs contains all emitted race control messages
	SessionInfo   json.RawMessage `json:"SessionInfo"`         // SessionInfo contains intrinsic data about the event and session
	SessionData   json.RawMessage `json:"SessionData"`         // SesionData contains all emitted session and track status changes
	TimingData    json.RawMessage `json:"TimingData"`          // TimingData represents driver-specific lap times, intervals, etc.
	LapCount      json.RawMessage `json:"LapCount"`            // LapCount contains the latest lap (current/total) data
}

// The heartbeat message indicates the client connection to the server is working even if there are
// no other messages coming from the server and keeps the websocket connection alive.
type heartbeat struct {
	ReceivedAt time.Time `json:"Utc"`
}

// timingAppData contains per-driver stint information, e.g. tire compound, stint length and driver
// position.
type timingAppData struct {
	Lines driverTimingAppMap `json:"Lines"`
}

// driverTimingAppList is a type allowing for custom ummarshaling of the driver timing app data
// which can include additional non-driver fields (e.g. _kf:true kvps).
type driverTimingAppMap map[string]drivingTimingAppData

func (dl *driverTimingAppMap) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	//
	filteredDriverList := make(map[string]drivingTimingAppData)
	for k, v := range m {
		if _, err := strconv.Atoi(k); err != nil {
			continue
		}
		var d drivingTimingAppData
		if err := json.Unmarshal(v, &d); err != nil {
			continue
		}
		filteredDriverList[k] = d
	}

	*dl = filteredDriverList
	return nil
}

// driverTimingAppData includes individual timing app data for a specific driver.
type drivingTimingAppData struct {
	RacingNumber string `json:"RacingNumber"`
	Line         *int   `json:"Line"`
	GridPos      string `json:"GridPos"`
	Stints       stints `json:"Stints"`
}

type stints map[string]stint

func (s *stints) UnmarshalJSON(data []byte) error {
	// first attempt to unmarshal change message structure (map)
	m := make(map[string]stint)
	if err := json.Unmarshal(data, &m); err == nil {
		*s = m
		return nil
	}
	// next attempt to unmarshal reference message structure (slice)
	var sl []stint
	if err := json.Unmarshal(data, &sl); err != nil {
		return err
	}
	for i, v := range sl {
		m[strconv.Itoa(i)] = v
	}
	*s = m
	return nil
}

type stint struct {
	LapFlags        *int    `json:"LapFlags"`
	Compound        *string `json:"Compound"`
	New             *string `json:"New"`
	TyresNotChanged *string `json:"TyresNotChanged"`
	TotalLaps       *int    `json:"TotalLaps"`
	StartLaps       *int    `json:"StartLaps"`
	LapTime         *string `json:"LapTime"`
	LapNumber       *int    `json:"LapNumber"`
}

type trackStatus struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
}

// driverList is a type allowing for custom ummarshaling of the driver list which can include
// additional non-driver fields (e.g. _kf:true kvps).
type driverList map[string]driverListItem

func (dl *driverList) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	filteredDriverList := make(map[string]driverListItem)
	for k, v := range m {
		if _, err := strconv.Atoi(k); err != nil {
			continue
		}
		var d driverListItem
		if err := json.Unmarshal(v, &d); err != nil {
			continue
		}
		filteredDriverList[k] = d
	}

	*dl = filteredDriverList
	return nil
}

// driverData represents intrinsic data about an individual driver
type driverListItem struct {
	RacingNumber  *string `json:"RacingNumber"`
	BroadcastName *string `json:"BroadcastName"`
	FullName      *string `json:"FullName"`
	ShortName     *string `json:"Tla"`
	Line          *int    `json:"Line"`
	TeamName      *string `json:"TeamName"`
	TeamColour    *string `json:"TeamColour"`
	FirstName     *string `json:"FirstName"`
	LastName      *string `json:"LastName"`
	Reference     *string `json:"Reference"`
	CountryCode   *string `json:"CountryCode"`
	HeadshotURL   *string `json:"HeadshotUrl"`
	NameFormat    *string `json:"NameFormat"`
}

// changeRaceCtrlMsgs contains a map of race control messages.
type changeRaceCtrlMsgs struct {
	Messages map[string]raceCtrlMsg `json:"Messages"`
}

// referenceRaceCtrlMsgs contains a list of race control messages.
type referenceRaceCtrlMsgs struct {
	Messages []raceCtrlMsg `json:"Messages"`
}

// raceCtrlMsgs represents a message or alert issued by Race Control. This includes information
// about investigations, penalties, track limits violations, flag information and more.
type raceCtrlMsg struct {
	UTC      string `json:"Utc"`
	Lap      int    `json:"Lap"`
	Category string `json:"Category"`
	Message  string `json:"Message"`
	Flag     string `json:"Flag"`
	Mode     string `json:"Mode"`
	Scope    string `json:"Scope"`
	Status   string `json:"Status"`
	Sector   int    `json:"Sector"`
}

// sessionInfo contains intrinsic data about the weekend event and current session. Typically this
// event is consumed as a part of the initial reference message without significant changes
// throughout the session
type sessionInfo struct {
	Meeting struct {
		Key          *int    `json:"Key"`
		Name         *string `json:"Name"`
		OfficialName *string `json:"OfficialName"`
		Location     *string `json:"Location"`
		Number       *int    `json:"Number"`
		Country      struct {
			Key  *int    `json:"Key"`
			Code *string `json:"Code"`
			Name *string `json:"Name"`
		} `json:"Country"`
		Circuit struct {
			Key       *int    `json:"Key"`
			ShortName *string `json:"ShortName"`
		} `json:"Circuit"`
	} `json:"Meeting"`
	ArchiveStatus struct {
		Status *string `json:"Status"`
	} `json:"ArchiveStatus"`
	Key       *int    `json:"Key"`
	Type      *string `json:"Type"`
	Number    *int    `json:"Number"`
	Name      *string `json:"Name"`
	StartDate *string `json:"StartDate"`
	EndDate   *string `json:"EndDate"`
	GMTOffset *string `json:"GMTOffset"`
	Path      *string `json:"Path"`
}

// sessionData contains session/track status changes. Change and Reference version of the message
// are identical except that the changes are represented in a map and the reference is represented
// as a list. This type handles unmarshaling both reference and change messages into a normalized
// structure
type sessionData struct {
	Series       map[string]sessionDataSeries       `json:"Series"`
	StatusSeries map[string]sessionDataStatusSeries `json:"StatusSeries"`
}

func (s *sessionData) UnmarshalJSON(data []byte) error {
	// first try unmarshalling change message version
	var change changeSessionData
	if err := json.Unmarshal(data, &change); err == nil {
		*s = sessionData(change)
		return nil
	}
	// if that fails try unmarshalling reference message version
	var ref referenceSessionData
	if err := json.Unmarshal(data, &ref); err != nil {
		return err
	}
	// convert array of session data into map
	dataMap := make(map[string]sessionDataSeries)
	for i, v := range ref.Series {
		dataMap[strconv.Itoa(i)] = v
	}
	s.Series = dataMap
	// convert array of session status data into map
	statusMap := make(map[string]sessionDataStatusSeries)
	for i, v := range ref.StatusSeries {
		statusMap[strconv.Itoa(i)] = v
	}
	s.StatusSeries = statusMap
	return nil
}

// referenceSessionData contains a slice of all session/track status changes and the corresponding
// lap in which the changes occurred (if the session is a race).
type referenceSessionData struct {
	Series       []sessionDataSeries       `json:"Series"` // Lap on which the status applies
	StatusSeries []sessionDataStatusSeries `json:"StatusSeries"`
}

// changeSessionData contains a map of all session/track status changes and the corresponding lap in
// which the changes occurred (if the session is  race).
type changeSessionData struct {
	Series       map[string]sessionDataSeries       `json:"Series"`
	StatusSeries map[string]sessionDataStatusSeries `json:"StatusSeries"`
}

// sessionDataSeries contains the lap count and qualifying part for session data messages.
type sessionDataSeries struct {
	UTC            time.Time `json:"Utc"`
	Lap            *int      `json:"Lap"`
	QualifyingPart *int      `json:"QualifyingPart"`
}

// sessionDataStatuseries contains a session and/or track status series change. These statuses
// include flags, (virtual) safety cards, etc.
type sessionDataStatusSeries struct {
	UTC           time.Time `json:"Utc"`
	TrackStatus   *string   `json:"TrackStatus"`
	SessionStatus *string   `json:"SessionStatus"`
}

// timingDataMsg represents per-driver live timing data including lap times, gaps, personal/
// overall best indicators and sector timing data.
type timingDataMsg struct {
	Lines driverTimingDataMap `json:"Lines"`
}

// drivertimingDataMap enables a custom json unmarshalling that removes non-drivertiming data from
// the map (e.g. _kf:true kvps)
type driverTimingDataMap map[string]driverTimingData

func (dt *driverTimingDataMap) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	filteredMap := make(map[string]driverTimingData)
	for k, v := range m {
		if _, err := strconv.Atoi(k); err != nil {
			continue
		}
		var d driverTimingData
		if err := json.Unmarshal(v, &d); err != nil {
			continue
		}
		filteredMap[k] = d
	}

	*dt = filteredMap
	return nil
}

// driverTimingData contains lap times, gaps and other live-timing information about a specific
// driver. Both `referenceDriverTimingData` and `changeDriverTimingData` 'inherit' the properties
// from `driverTimingData`
type driverTimingData struct {
	Line         *int    `json:"Line"`
	Position     *string `json:"Position"`     // current position on timing board
	ShowPosition *bool   `json:"ShowPosition"` // Will be false when a driver is out of the session (race), or out of the session (qualifying)
	RacingNumber *string `json:"RacingNumber"` // the unique driver number
	Retired      *bool   `json:"Retired"`      // car and driver have retired from the race
	InPit        *bool   `json:"InPit"`        // car is in pit
	PitOut       *bool   `json:"PitOut"`       // current lap is an out-lap
	Stopped      *bool   `json:"Stopped"`      // true when car is not moving
	// Statuses:
	Status                  *int                 `json:"Status"`
	GapToLeader             *string              `json:"GapToLeader"`
	IntervalToPositionAhead driverTimingInterval `json:"IntervalToPositionAhead"`
	Speeds                  driverTimingSpeeds   `json:"Speeds"`
	BestLapTime             driverTimingBestLap  `json:"BestLapTime"`
	LastLapTime             driverTimingLastLap  `json:"LastLapTime"`
	NumberOfLaps            *int                 `json:"NumberOfLaps"`
	KnockedOut              *bool                `json:"KnockedOut"`
	Cutoff                  *bool                `json:"Cutoff"`
	Sectors                 driverTimingSectors  `json:"Sectors"`
	QualifyingStats         driverTimingStats    `json:"Stats"`
	QualifyingBestLapTimes  driverTimingBestLaps `json:"BestLapTimes"`
}

type driverTimingInterval struct {
	Value    *string `json:"Value"`
	Catching *bool   `json:"Catching"`
}

type driverTimingSpeeds struct {
	FirstIntermediatePoint  driverSpeedTimingData `json:"I1"`
	SecondIntermediatePoint driverSpeedTimingData `json:"I2"`
	SpeedTrap               driverSpeedTimingData `json:"ST"`
}

type driverTimingBestLap struct {
	Value *string `json:"Value"`
	Lap   *int    `json:"Lap"`
}

type driverTimingLastLap struct {
	Value           *string `json:"Value"`
	Status          *int    `json:"Status"`
	OverallFastest  *bool   `json:"OverallFastest"`
	PersonalFastest *bool   `json:"PersonalFastest"`
}

// driverSpeedTimingData represents speed-trap-like data captured at various points around the
// circuit for a specific driver on a particular lap.
type driverSpeedTimingData struct {
	Value           *string `json:"Value"`
	Status          *int    `json:"Status"`
	OverallFastest  *bool   `json:"OverallFastest"`
	PersonalFastest *bool   `json:"PersonalFastest"`
}

// driverTimingSectors represents per-sector timing data; Change and Reference version of the
// message are identical except that the changes are represented in a map and the reference is
// represented as a list. This type handles unmarshaling both reference and change messages into a
// normalized structure.
type driverTimingSectors map[string]sectorTiming

func (dts *driverTimingSectors) UnmarshalJSON(data []byte) error {
	// first try unmarshalling change message structure
	m := make(map[string]sectorTiming)
	if err := json.Unmarshal(data, &m); err == nil {
		*dts = m
		return nil
	}
	// next try unmarshalling reference message structure
	var s []sectorTiming
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// convert slice to map
	for i, v := range s {
		m[strconv.Itoa(i)] = v
	}
	*dts = m
	return nil
}

// driverTimingSectors represents per-sector timing data; Change and Reference version of the
// message are identical except that the changes are represented in a map and the reference is
// represented as a list. This type handles unmarshaling both reference and change messages into a
// normalized structure.
type driverTimingStats map[string]driverQualifyingTimingStat

func (dts *driverTimingStats) UnmarshalJSON(data []byte) error {
	// first try unmarshalling change message structure
	var m map[string]driverQualifyingTimingStat
	if err := json.Unmarshal(data, &m); err == nil {
		*dts = m
		return nil
	}
	// next try unmarshalling reference message structure
	var s []driverQualifyingTimingStat
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// convert slice to map
	m = make(map[string]driverQualifyingTimingStat)
	for i, v := range s {
		m[strconv.Itoa(i)] = v
	}
	*dts = m
	return nil
}

type driverTimingBestLaps map[string]driverTimingBestLap

func (dts *driverTimingBestLaps) UnmarshalJSON(data []byte) error {
	// first try unmarshalling change message structure
	var m map[string]driverTimingBestLap
	if err := json.Unmarshal(data, &m); err == nil {
		*dts = m
		return nil
	}
	// next try unmarshalling reference message structure
	var s []driverTimingBestLap
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// convert slice to map
	m = make(map[string]driverTimingBestLap)
	for i, v := range s {
		m[strconv.Itoa(i)] = v
	}
	*dts = m
	return nil
}

type driverQualifyingTimingStat struct {
	TimeDiffToFastest       *string `json:"TimeDiffToFastest"`
	TimeDiffToPositionAhead *string `json:"TimeDiffToPositionAhead"`
}

// sectorTiming represents timing for 1 of 3 sectors around the crcuit for a specific driver on a
// particular lap.
type sectorTiming struct {
	Stopped       *bool                `json:"Stopped"`
	Value         *string              `json:"Value"`
	Status        *int                 `json:"Status"`
	OverallBest   *bool                `json:"OverallFastest"`
	PersonalBest  *bool                `json:"PersonalFastest"`
	PreviousValue *string              `json:"PreviousValue"`
	Segments      segmentTimingDataMap `json:"Segments"`
}

// segmentTimingDataMap is a type alias for a map of segment timing data items that allows for
// custom unmarshalling logic to handle the different structures between reference and change
// messages.
type segmentTimingDataMap map[string]segmentTimingData

func (stdm *segmentTimingDataMap) UnmarshalJSON(data []byte) error {
	m := make(map[string]segmentTimingData)
	// first try unmarshalling change message structure
	if err := json.Unmarshal(data, &m); err == nil {
		*stdm = m
		return nil
	}
	// next try unmarshalling reference message structure
	s := make([]segmentTimingData, 0, 5)
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// convert slice to map
	for i, v := range s {
		m[strconv.Itoa(i)] = v
	}
	*stdm = m
	return nil
}

type segmentTimingData struct {
	Status *int `json:"Status"`
}

// lapCount represents the latest lap information of the session, including the `CurrentLap` of the
// leader in races.
type lapCount struct {
	CurrentLap *int `json:"CurrentLap"`
	TotalLaps  *int `json:"TotalLaps"`
}
