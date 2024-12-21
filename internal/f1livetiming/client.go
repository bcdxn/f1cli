package f1livetiming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/coder/websocket"
	"github.com/qdm12/reprint"
)

// New returns a new F1 LiveTiming API Client.
func New(opts ...ClientOption) Client {
	// create a default instance of the client
	c := Client{
		drivers:       make(map[string]domain.Driver),
		meeting:       domain.NewMeeting(),
		driversCh:     make(chan map[string]domain.Driver),
		meetingCh:     make(chan domain.Meeting),
		raceCtrlMsgCh: make(chan domain.RaceCtrlMsg),
		doneCh:        make(chan error),
		logger:        slog.Default(),
		httpBaseURL:   "http://localhost:3000",
		wsBaseURL:     "ws://localhost:3000",
	}
	// apply given options
	for _, opt := range opts {
		opt(&c)
	}
	// return new instance of the client
	return c
}

type Client struct {
	// Internal Session State
	drivers         map[string]domain.Driver
	meeting         domain.Meeting
	raceCtrlMsg     domain.RaceCtrlMsg
	connectionToken string
	cookie          string
	// channels
	driversCh     chan map[string]domain.Driver
	meetingCh     chan domain.Meeting
	raceCtrlMsgCh chan domain.RaceCtrlMsg
	doneCh        chan error
	// F1 Live Timing API Configuration
	httpBaseURL string
	wsBaseURL   string
	// logger
	logger *slog.Logger
}

/* Client Optional Functional Parameters
------------------------------------------------------------------------------------------------- */

type ClientOption = func(c *Client)

// WithHTTPBaseURL configures the HTTP(S) URL of the F1 LiveTiming API; primarily used for testing.
func WithHTTPBaseURL(baseUrl string) ClientOption {
	return func(c *Client) { c.httpBaseURL = baseUrl }
}

// WithWSBaseURL configures the websocket URL of the F1 LiveTiming API; primarily used for
// testing.
func WithWSBaseURL(baseUrl string) ClientOption {
	return func(c *Client) { c.wsBaseURL = baseUrl }
}

// WithLogger configures the logger to use within the client.
func WithLogger(l *slog.Logger) ClientOption {
	return func(c *Client) { c.logger = l }
}

/* Client API
------------------------------------------------------------------------------------------------- */

// DriversCh exposes the drivers channel as read-only; a full snapshot of the drivers' intrinsic
// data and timing data can be read from this channel on each update from the F1 LiveTiming API.
func (c Client) Drivers() <-chan map[string]domain.Driver {
	return c.driversCh
}

// MeetingCh exposes the meeting channel as read-only; a full snapshot of the meeting and current
// session data can be read from this channel on each update from the F1 LiveTiming API.
func (c Client) Meeting() <-chan domain.Meeting {
	return c.meetingCh
}

// RaceCtrlMsgsCh exposes the race control messages channel as read-only; a full list of all race
// control messages can be read from this channel on each update from the F1 LiveTiming API.
func (c Client) RaceCtrlMsgs() <-chan domain.RaceCtrlMsg {
	return c.raceCtrlMsgCh
}

// DoneCh allows the client to signal to the caller that it has exited; this can happen if an error
// occurs or if the websocket connection is closed by the server.
func (c Client) Done() <-chan error {
	return c.doneCh
}

func (c *Client) Listen(ctx context.Context) {
	defer close(c.doneCh)
	// Call negotiate to get required token/cookie values
	c.negotiate()
	// Derive the websocket URL
	u, err := c.websocketURL()
	if err != nil {
		c.logger.Error("error building websocket URL")
		c.doneCh <- err
		close(c.doneCh)
		return
	}
	// Add required headers
	headers := make(http.Header)
	headers.Add("User-Agent", "BestHTTP")
	headers.Add("Accept-Encoding", "gzip,identity")
	headers.Add("Cookie", c.cookie)
	// Create the websocket connection with the F1 livetiming API server
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		c.logger.Error("error dialing websocket", "err", err.Error())
		c.doneCh <- err
		close(c.doneCh)
		return
	}
	defer conn.CloseNow()
	// disable size limitats as the F1 LiveTiming API sends some big messages
	conn.SetReadLimit(-1)
	// send subscribe message to start receiving messages from the F1 LiveTiming API
	err = c.sendSubscribeMsg(conn)
	if err != nil {
		c.doneCh <- err
		close(c.doneCh)
		return
	}

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				conn.Close(websocket.StatusNormalClosure, "client closed")
			} else {
				c.doneCh <- err
			}
			return
		}
		// No errors, process the message from the livetiming API
		c.processMessage(msg)
	}
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

// negotiate calls the F1 LiveTiming API, retreiving information required to start the websocket
// connection required to receive real-time updates.
func (c *Client) negotiate() error {
	req, err := c.negotiateRequest()
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending f1 livetiming api negotiation request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		ct, err := c.parseConnectionToken(resp.Body)
		if err != nil {
			return fmt.Errorf("error parsing connection token: %w", err)
		}
		c.connectionToken = ct
		c.cookie = resp.Header.Get("set-cookie")
		c.logger.Debug("successfully negotiated connection; connection token len:", "token_length", len(ct))
		return nil
	default:
		return fmt.Errorf("error negotiating f1 livetiming api connection: %w", errors.New(resp.Status))
	}
}

// negotiateRequest creates the HTTP request object that is required to initiate the connection to
// the F1 Live Timing Signalr API.
func (c Client) negotiateRequest() (*http.Request, error) {
	var r *http.Request
	u, err := url.Parse(c.httpBaseURL)
	if err != nil {
		return r, fmt.Errorf("invalid HTTPBaseURL: %w", err)
	}

	r = &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: u.Scheme,
			Host:   u.Host,
			Path:   "/signalr/negotiate",
			RawQuery: url.Values{
				"connectionData": {`[{"Name":"Streaming"}]`},
				"clientProtocol": {"1.5"},
			}.Encode(),
		},
	}

	return r, nil
}

// sendSubscribeMsg sends a message that tells the server which types of data messages we would like
// to receive as required by the F1 Live Timing API.
func (Client) sendSubscribeMsg(conn *websocket.Conn) error {
	return conn.Write(context.Background(), websocket.MessageText, []byte(`
      {
          "H": "Streaming",
          "M": "Subscribe",
          "A": [[
              "Heartbeat",
              "TimingStats",
              "TimingAppData",
              "TrackStatus",
              "DriverList",
              "RaceControlMessages",
              "SessionInfo",
              "SessionData",
              "LapCount",
              "TimingData"
          ]],
          "I": 1
      }
  `))
}

// parseConnectionToken is a helper function that parses the negotiate response pulling out the
// connectionToken field from the body. This token is required in the subsequent connect request
// that creates the websocket connection.
func (Client) parseConnectionToken(body io.ReadCloser) (string, error) {
	var n negotiateResponse
	var t string

	b, err := io.ReadAll(body)
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(b, &n)
	if err != nil {
		return t, err
	}

	return n.ConnectionToken, nil
}

// websocketURL is a helper method that generates the URL with appropriate query parameters
// required to start the websocket connection.
func (c Client) websocketURL() (*url.URL, error) {
	var u *url.URL
	u, err := url.Parse(c.wsBaseURL)
	if err != nil {
		return u, fmt.Errorf("invalid HTTPBaseURL: %w", err)
	}

	u = &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   "/signalr/connect",
		RawQuery: url.Values{
			"connectionData":  {`[{"Name":"Streaming"}]`},
			"connectionToken": {c.connectionToken},
			"clientProtocol":  {"1.5"},
			"transport":       {"webSockets"},
		}.Encode(),
	}

	return u, nil
}

// processMessage checks the message coming the F1 LiveTiming Client to see if it is a 'change'
// message or a 'reference' message and handles them appropriately, transforming the message into
// 1 to none or many messages that can be written to the client channels.
func (c *Client) processMessage(msg []byte) {
	// Always try to parse a change message first since there is only 1 reference message and
	// tens of thousands of change messages over the course of a session
	var f1msg f1Message
	err := json.Unmarshal(msg, &f1msg)
	if err != nil {
		c.logger.Warn("unknown message format", "msg", string(msg), "err", err)
		return
	}

	if len(f1msg.Changes) > 0 {
		c.logger.Debug("received change data message")
		c.processChangeMessage(f1msg.Changes)
	}

	if len(f1msg.Reference) > 0 {
		c.logger.Debug("received reference data message")
		c.processReferenceMessage(f1msg.Reference)
		return
	}
}

// processChangeMessage handles an incoming change message from the F1 Live Timing API; change
// messages represent deltas to the original reference message and all preceeding change messages.
// Once processed, a simplified event is emitted for use by the TUI.
func (c *Client) processChangeMessage(changesRawMessage []byte) {
	var changesMsg []f1ChangeMessage
	err := json.Unmarshal(changesRawMessage, &changesMsg)
	if err != nil {
		c.logger.Warn("error unmarshalling change message", "msg", string(changesRawMessage), "err", err)
		return
	}
	meetingUpdating := false
	driversUpdated := false
	raceCtrlMsgsUpdated := false
	for _, m := range changesMsg {
		if len(m.Arguments) == 3 {
			var s, d, r bool
			var msgType string
			err := json.Unmarshal(m.Arguments[0], &msgType)
			if err != nil {
				c.logger.Warn("invalid message type argument", "arg", string(m.Arguments[0]))
				continue
			}
			msgData := m.Arguments[1]

			switch msgType {
			case "DriverList":
				s, d, r = c.updateDriverList(c.unmarshalDriverListMsg(msgData))
			case "TimingData":
				s, d, r = c.updateTimingData(c.unmarshalTimingDataMsg(msgData))
			case "SessionInfo":
				s, d, r = c.updateSessionInfo(c.unmarshalSessionInfoMsg(msgData))
			case "SessionData":
				s, d, r = c.updateSessionData(c.unmarshalSessionDataMsg(msgData))
			case "LapCount":
				s, d, r = c.updateLapCount(c.unmarshalLapCountMsg(msgData))
			case "TimingAppData":
				s, d, r = c.updateTimingAppData(c.unmarshalTimingAppDataMsg(msgData))
			case "RaceControlMessages":
				s, d, r = c.updateRaceCtrlMsg(c.unmarshalRaceCtrlMsg(msgData))
			default:
				c.logger.Warn("unknown change message", "type", msgType, "msg", string(msgData))
			}

			if s {
				meetingUpdating = true
			}
			if d {
				driversUpdated = true
			}
			if r {
				raceCtrlMsgsUpdated = true
			}
		} else {
			c.logger.Warn("invalid length of change message arguments", "args", m.Arguments)
		}
	}

	if meetingUpdating {
		c.writeMeetingToChan()
	}
	if driversUpdated {
		c.writeDriversToChan()
	}
	if raceCtrlMsgsUpdated {
		c.writeRaceCtrlMsgsToChan()
	}
}

func (c *Client) processReferenceMessage(referenceRawMsg []byte) {
	var refMsg f1ReferenceMessage
	err := json.Unmarshal(referenceRawMsg, &refMsg)
	if err != nil {
		c.logger.Warn("error unmarshalling reference message", "msg", string(referenceRawMsg), "err", err)
		return
	}

	c.updateSessionInfo(c.unmarshalSessionInfoMsg(refMsg.SessionInfo))
	c.updateSessionData(c.unmarshalSessionDataMsg(refMsg.SessionData))
	c.updateDriverList(c.unmarshalDriverListMsg(refMsg.DriverList))
	c.updateLapCount(c.unmarshalLapCountMsg(refMsg.LapCount))
	c.updateTimingData(c.unmarshalTimingDataMsg(refMsg.TimingData))
	c.updateTimingAppData(c.unmarshalTimingAppDataMsg(refMsg.TimingAppData))
	c.updateRaceCtrlMsg(c.unmarshalRaceCtrlMsg(refMsg.RaceCtrlMsgs))
	// The reference message always updates all channels
	c.writeMeetingToChan()
	c.writeDriversToChan()
	c.writeRaceCtrlMsgsToChan()
}

/* Message Unmarshalers
------------------------------------------------------------------------------------------------- */

const (
	f1APIDateLayout = "2006-01-02T15:04:05-0700" // date format used by the F1 LiveTiming API
)

// unmarshalSessionInfo converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalSessionInfoMsg(msg []byte) sessionInfo {
	var s sessionInfo
	err := json.Unmarshal(msg, &s)
	if err != nil {
		c.logger.Warn("session info msg in unknown format", "msg", string(msg))
	}
	return s
}

// unmarshalSessionData converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalSessionDataMsg(msg []byte) sessionData {
	var s sessionData
	err := json.Unmarshal(msg, &s)
	if err != nil {
		c.logger.Warn("session data msg in unknown format", "msg", string(msg))
	}
	return s
}

// ummarshalDriverListMsg converts the websocket message to a strongly typed map of structs.
func (c *Client) unmarshalDriverListMsg(msg []byte) driverList {
	var drivers driverList
	err := json.Unmarshal(msg, &drivers)
	if err != nil {
		c.logger.Warn("driver data msg in unknown format", "msg", string(msg))
	}

	return drivers
}

// unmarshalLapCountMsg converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalLapCountMsg(msg []byte) lapCount {
	var lc lapCount
	err := json.Unmarshal(msg, &lc)
	if err != nil {
		c.logger.Warn("lap count msg in unknown format", "msg", string(msg))
	}
	return lc
}

// unmarshalTimingDataMsg converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalTimingDataMsg(msg []byte) timingDataMsg {
	var timingDataMsg timingDataMsg
	err := json.Unmarshal(msg, &timingDataMsg)
	if err != nil {
		c.logger.Warn("timing data msg in unknown format", "msg", string(msg))
	}

	return timingDataMsg
}

// unmarshalTimingAppDataMsg converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalTimingAppDataMsg(msg []byte) timingAppData {
	var tad timingAppData
	err := json.Unmarshal(msg, &tad)
	if err != nil {
		c.logger.Warn("timing app data msg in unknown format", "msg", string(msg))
	}

	return tad
}
func (c *Client) unmarshalRaceCtrlMsg(msg []byte) raceCtrlMsgs {
	var rcm raceCtrlMsgs
	err := json.Unmarshal(msg, &rcm)
	if err != nil {
		c.logger.Warn("race ctrl msg in unknown format", "msg", string(msg))
	}

	return rcm
}

/* Channel Updaters
------------------------------------------------------------------------------------------------- */

// updateSessionInfo converts a SessionInfo msg from the F1 Live Timing API to the `Meeting` and
// `Session` domain models  stored in the client's internal state store and writes the full state
// of the meeting for consumers to read.
func (c *Client) updateSessionInfo(session sessionInfo) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always updates session
	meetingUpdating = true
	setMeetingName(&c.meeting, session.Meeting.Name)
	setMeetingFullName(&c.meeting, session.Meeting.OfficialName)
	setMeetingLocation(&c.meeting, session.Meeting.Location)
	setMeetingRoundNumber(&c.meeting, session.Meeting.Number)
	setMeetingCountryCode(&c.meeting, session.Meeting.Country.Code)
	setMeetingCountryName(&c.meeting, session.Meeting.Country.Name)
	setMeetingCurcuitShortName(&c.meeting, session.Meeting.Circuit.ShortName)
	setSessionName(&c.meeting, session.Name)
	setSessionGMTOffset(&c.meeting, session.GMTOffset)
	setSessionStartDate(&c.meeting, session.StartDate)
	setSessionEndDate(&c.meeting, session.EndDate)
	setSessionType(&c.meeting, session.Type)
	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

// updateSessionData converts a SessionData msg from the F1 LiveTiming API to the `Session` domain
// model and writes the full state of the meeting/session for consumers to read.
func (c *Client) updateSessionData(session sessionData) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always updates session
	meetingUpdating = true
	// update the session status to the latest/current status
	statusKeys := make([]string, 0)
	for key := range session.StatusSeries {
		statusKeys = append(statusKeys, key)
	}
	// Access the status messages in order so that we end up on the latest entry
	sort.Strings(statusKeys)
	for _, key := range statusKeys {
		setSessionStatus(&c.meeting, session.StatusSeries[key].SessionStatus)
	}

	// Update the session part to the latest/current session part
	seriesKeys := make([]string, 0)
	for key := range session.Series {
		seriesKeys = append(seriesKeys, key)
	}
	// Access the series messages in order so that we end up on the latest entry
	sort.Strings(seriesKeys)
	for _, key := range seriesKeys {
		setSessionPart(&c.meeting, session.Series[key].QualifyingPart)
	}

	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

// updateDriverIntrinsicData updates the intrinsic driver data (and occassionally position).
func (c *Client) updateDriverList(driverDataMsg map[string]driverListItem) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always updates drivers
	driversUpdated = true
	// update data for each driver to the drivers map
	for number, data := range driverDataMsg {
		// retrieve existing driver data from the map if it exists or create a new driver
		driver, ok := c.drivers[number]
		if !ok {
			driver = domain.NewDriver(number)
		}
		// Overwrite fields
		setShortName(&driver, data.ShortName)
		setDriverName(&driver, data.FirstName, data.LastName, data.NameFormat)
		setTeamName(&driver, data.TeamName)
		setTeamColor(&driver, data.TeamColour)
		setPosition(&driver, data.Line)
		// write the driver data back to the client state store
		c.drivers[number] = driver
	}
	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

// updateLapCount updates the current/total lap data (only applicable during races).
func (c *Client) updateLapCount(lc lapCount) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always update the session
	meetingUpdating = true
	if lc.CurrentLap != nil {
		c.meeting.Session.CurrentLap = *lc.CurrentLap
	}
	if lc.TotalLaps != nil {
		c.meeting.Session.TotalLaps = *lc.TotalLaps
	}

	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

// updateDriverTimingData updates driver timing and position data.
func (c *Client) updateTimingData(timingDataMsg timingDataMsg) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always updates drivers
	driversUpdated = true
	// add data for each driver to the drivers map
	for number, data := range timingDataMsg.Lines {
		// retrieve existing driver data from the map if it exists or create a new driver
		driver, ok := c.drivers[number]
		if !ok {
			driver = domain.NewDriver(number)
		}
		// Overwrite fields
		setPosition(&driver, data.Line)
		setGaps(&driver, c.meeting, data)
		setLastLap(&driver, data.LastLapTime.Value, data.LastLapTime.PersonalFastest)
		if data.LastLapTime.OverallFastest != nil && *data.LastLapTime.OverallFastest {
			c.meeting.Session.FastestLapOwner = number
			meetingUpdating = true
		}
		setBestLap(&driver, data.BestLapTime.Value)
		setIsKnockedOut(&driver, data.KnockedOut)
		setIsRetired(&driver, data.Retired, data.Status)
		setNumberOfLaps(&driver, data.NumberOfLaps)
		if updated := setSectors(&driver, c.meeting, data.Sectors); updated {
			meetingUpdating = true
		}
		// Set the Pit status _after_ setting sectors, because these functions may overwrite sector
		// data to prevent weird scenarios of having sector times posted while in the PIT or Outlap
		setIsInPit(&driver, data.InPit)
		setIsPitOut(&driver, data.PitOut)
		// keep track of best lap times in each qualifying part
		setBestLapInPart(&driver, data)

		// update the driver data in the map
		c.drivers[number] = driver
	}

	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

// updateTimingAppData updates driver stint and position data.
func (c *Client) updateTimingAppData(tad timingAppData) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always updates drivers
	driversUpdated = true
	for driverNum, timingAppData := range tad.Lines {
		// if multiple stints are given (e.g. in the reference message) we'll iterate through them,
		// taking the stint with the largest key (which are numbers indicating the order)
		stints := make([]string, 0)
		for stintNum := range timingAppData.Stints {
			stints = append(stints, stintNum)
		}
		if len(stints) == 0 {
			continue
		}
		// sort the stints in descending order by key so we can take the largest key at index 0
		sort.Slice(stints, func(i, j int) bool {
			return stints[i] > stints[j]
		})
		currentStint := stints[0]

		driver, ok := c.drivers[driverNum]
		if !ok {
			c.logger.Error("driver not found", "num", driverNum)
			driver = domain.NewDriver(driverNum)
		}
		if len(timingAppData.Stints) > 0 {
			setTireCompound(&driver, timingAppData.Stints[currentStint].Compound)
			setTireLapCount(&driver, timingAppData.Stints[currentStint].TotalLaps)
		}
		// TimingAppData also contains driver position data sometimes
		setPosition(&driver, timingAppData.Line)
		// overwrite the driver state with the new stint information
		c.drivers[driverNum] = driver
	}

	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

func (c *Client) updateRaceCtrlMsg(msgs raceCtrlMsgs) (meetingUpdating, driversUpdated, raceCtrlMsgsUpdated bool) {
	// this function always updates race control messages
	raceCtrlMsgsUpdated = true
	// get the latest message by sorting the keys
	var latestMsg raceCtrlMsg
	rcmKeys := make([]string, 0)
	for key := range msgs.Messages {
		rcmKeys = append(rcmKeys, key)
	}
	sort.Strings(rcmKeys)
	for _, key := range rcmKeys {
		latestMsg = msgs.Messages[key]
	}

	c.raceCtrlMsg = domain.RaceCtrlMsg{
		Body: latestMsg.Message,
	}

	switch latestMsg.Category {
	case raceCtrlStatusFlag:
		c.raceCtrlMsg.Category = domain.RaceCtrlMsgCategoryTrackStatus
		switch latestMsg.Flag {
		case raceCtrlFlagClear:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagGreen
		case raceCtrlFlagGreen:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagGreen
		case raceCtrlFlagBlue:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagBlue
		case raceCtrlFlagYellow:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagYellow
		case raceCtrlFlagDoubleYellow:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagDoubleYellow
		case raceCtrlFlagRed:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagRed
		case raceCtrlFlagBW:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFlagBW
		default:
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleDefault
		}
	case raceCtrlStatusSC:
		c.raceCtrlMsg.Category = domain.RaceCtrlMsgCategoryTrackStatus
		if latestMsg.Mode == raceCtrlModeSC {
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleSC
		} else if latestMsg.Mode == raceCtrlModeVSC {
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleVSC
		} else {
			c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleDefault
		}
	case raceCtrlStatusDRS:
		c.raceCtrlMsg.Category = domain.RaceCtrlMsgCategoryFIA
		c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleDefault
	case raceCtrlStatusOther:
		c.raceCtrlMsg.Category = domain.RaceCtrlMsgCategoryFIA
		c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleFIA
	default:
		c.raceCtrlMsg.Category = domain.RaceCtrlMsgCategoryOther
		c.raceCtrlMsg.Title = domain.RaceCtrlMsgTitleDefault
	}

	return meetingUpdating, driversUpdated, raceCtrlMsgsUpdated
}

// writeMeetingToChan writes  a copy of the meeting to ensure concurrency safety between goroutines.
func (c *Client) writeMeetingToChan() {
	var cpy domain.Meeting

	reprint.FromTo(&c.meeting, &cpy)

	c.meetingCh <- cpy
}

// Because maps are not concurrency-safe, we'll copy the map before writing it to the channel that
// can be read by concurrent goroutines.
func (c *Client) writeDriversToChan() {
	var cpy map[string]domain.Driver

	reprint.FromTo(&c.drivers, &cpy)

	c.driversCh <- cpy
}

// Because slices are not concurrency-safe, we'll copy the slice before writing it to the channel
// that can be read by concurrent goroutines.
func (c *Client) writeRaceCtrlMsgsToChan() {
	var cpy domain.RaceCtrlMsg
	reprint.FromTo(&c.raceCtrlMsg, &cpy)
	c.raceCtrlMsgCh <- cpy
}

/* Message Transformers
------------------------------------------------------------------------------------------------- */

func setShortName(driver *domain.Driver, shortName *string) {
	if shortName != nil {
		driver.ShortName = *shortName
	}
}

func setDriverName(driver *domain.Driver, firstName, lastName, nameFormat *string) {
	if firstName != nil && lastName != nil {
		if nameFormat != nil && *nameFormat == "LastNameIsPrimary" {
			driver.Name = *lastName + " " + *firstName
		} else {
			driver.Name = *firstName + " " + *lastName
		}
	}
}

func setTeamName(driver *domain.Driver, name *string) {
	if name != nil {
		driver.TeamName = *name
	}
}

func setTeamColor(driver *domain.Driver, color *string) {
	if color != nil {
		driver.TeamColor = "#" + *color
	}
}

func setPosition(driver *domain.Driver, pos *int) {
	if pos != nil {
		driver.TimingData.Position = *pos
	}
}

func setGaps(driver *domain.Driver, meeting domain.Meeting, data driverTimingData) {
	if driver.TimingData.Position == 1 {
		driver.TimingData.IntervalGap = ""
		driver.TimingData.LeaderGap = ""
	} else if meeting.Session.Type == domain.SessionTypeQualifying {
		// In Qualifying Sessions the interval is stored separately for each qualifying part; we're only
		// interested the most recent qualifying part, so we iterate through (the list is in order) and
		// overwrite the gaps for each available qualifying part ending with the most recent.
		parts := make([]string, 0, 3)
		for part := range data.QualifyingStats {
			parts = append(parts, part)
		}
		sort.Strings(parts)
		for _, part := range parts {
			if data.QualifyingStats[part].TimeDiffToFastest != nil && *data.QualifyingStats[part].TimeDiffToFastest != "" {
				driver.TimingData.LeaderGap = *data.QualifyingStats[part].TimeDiffToFastest
			}
			if data.QualifyingStats[part].TimeDiffToPositionAhead != nil && *data.QualifyingStats[part].TimeDiffToPositionAhead != "" {
				driver.TimingData.IntervalGap = *data.QualifyingStats[part].TimeDiffToPositionAhead
			}
		}
	} else {
		if data.IntervalToPositionAhead.Value != nil && *data.IntervalToPositionAhead.Value != "" {
			driver.TimingData.IntervalGap = *data.IntervalToPositionAhead.Value
		}
		if data.GapToLeader != nil && *data.GapToLeader != "" {
			driver.TimingData.LeaderGap = *data.GapToLeader
		}
	}
}

func setLastLap(driver *domain.Driver, time *string, personalFastest *bool) {
	if time != nil && *time != "" {
		driver.TimingData.LastLap.Time = *time
	}

	if personalFastest != nil {
		driver.TimingData.LastLap.IsPersonalBest = *personalFastest
	}
}

func setBestLap(driver *domain.Driver, time *string) {
	if time != nil && *time != "" {
		driver.TimingData.BestLapTime = *time
	}
}

func setIsKnockedOut(driver *domain.Driver, out *bool) {
	if out != nil {
		driver.TimingData.IsKnockedOut = *out
	}
}

func setIsRetired(driver *domain.Driver, out *bool, status *int) {
	if out != nil {
		driver.TimingData.IsRetired = *out
	}

	if status != nil &&
		(*status == statusCrashDamageRetiredInPit ||
			*status == statusCrashDamageRetiredOnTrack) {
		driver.TimingData.IsRetired = true
	}
}

func setTireCompound(driver *domain.Driver, compound *string) {
	if compound != nil {
		switch *compound {
		case "SOFT":
			driver.TimingData.TireCompound = domain.TireCompoundSoft
		case "MEDIUM":
			driver.TimingData.TireCompound = domain.TireCompoundMedium
		case "HARD":
			driver.TimingData.TireCompound = domain.TireCompoundHard
		case "INTERMEDIATE":
			driver.TimingData.TireCompound = domain.TireCompoundIntermediate
		case "WET":
			driver.TimingData.TireCompound = domain.TireCompoundFullWet
		case "TEST":
			driver.TimingData.TireCompound = domain.TireCompoundTest
		case "PROTOTYPE":
			driver.TimingData.TireCompound = domain.TireCompoundTest
		default:
			driver.TimingData.TireCompound = domain.TireCompoundUnknown
		}
	}
}

func setTireLapCount(driver *domain.Driver, count *int) {
	if count != nil {
		driver.TimingData.TireLapCount = *count
	}
}

func setNumberOfLaps(driver *domain.Driver, laps *int) {
	if laps != nil {
		driver.TimingData.NumberOfLaps = *laps
	}
}

func setIsInPit(driver *domain.Driver, pit *bool) {
	if pit != nil {
		driver.TimingData.IsInPit = *pit
	}
}

func setIsPitOut(driver *domain.Driver, out *bool) {
	if out != nil {
		driver.TimingData.IsPitOut = *out
	}
}

func setSectors(driver *domain.Driver, meeting domain.Meeting, sectors map[string]sectorTiming) bool {
	for sectorNum, secData := range sectors {
		sector, ok := driver.TimingData.Sectors[sectorNum]
		if !ok {
			sector = domain.NewSector()
		}
		for segmentNum, segData := range secData.Segments {
			segment, ok := sector.Segments[segmentNum]
			if !ok {
				segment = domain.Segment{}
			}
			if meeting.Session.Status != domain.SessionStatusStarted {
				segment.Status = domain.SectorStatusInactive
				sector.Segments[segmentNum] = segment
			} else if segData.Status != nil {
				// convert f1 livetiming status to domain model status
				switch *segData.Status {
				case yellowSegment:
					segment.Status = domain.SectorStatusNotPersonalBest
				case greenSegment:
					segment.Status = domain.SectorStatusPersonalBest
				case purpleSegment:
					segment.Status = domain.SectorStatusOverallBest
				case pitSegment:
					segment.Status = domain.SectorStatusInactive
				default:
					segment.Status = domain.SectorStatusInactive
				}
				sector.Segments[segmentNum] = segment
			}
		}
		driver.TimingData.Sectors[sectorNum] = sector
	}
	return false
}

func setBestLapInPart(driver *domain.Driver, data driverTimingData) {
	// Sort session parts before
	partNums := make([]string, 0, 3)
	for partNum := range data.QualifyingBestLapTimes {
		partNums = append(partNums, partNum)
	}
	sort.Strings(partNums)
	for _, partNum := range partNums {
		i, _ := strconv.Atoi(partNum)
		if data.QualifyingBestLapTimes[partNum].Value != nil {
			driver.TimingData.BestLapTimes[i] = *data.QualifyingBestLapTimes[partNum].Value
		}
	}
}

func setMeetingName(meeting *domain.Meeting, name *string) {
	if name != nil {
		meeting.Name = *name
	}
}

func setMeetingFullName(meeting *domain.Meeting, name *string) {
	if name != nil {
		meeting.FullName = *name
	}
}

func setMeetingLocation(meeting *domain.Meeting, loc *string) {
	if loc != nil {
		meeting.Location = *loc
	}
}

func setMeetingRoundNumber(meeting *domain.Meeting, num *int) {
	if num != nil {
		meeting.RoundNumber = *num
	}
}

func setMeetingCountryCode(meeting *domain.Meeting, cc *string) {
	if cc != nil {
		meeting.CountryCode = *cc
	}
}

func setMeetingCountryName(meeting *domain.Meeting, name *string) {
	if name != nil {
		meeting.CountryName = *name
	}
}

func setMeetingCurcuitShortName(meeting *domain.Meeting, name *string) {
	if name != nil {
		meeting.CircuitShortName = *name
	}
}

func setSessionName(meeting *domain.Meeting, name *string) {
	if name != nil {
		meeting.Session.Name = *name
	}
}

func setSessionGMTOffset(meeting *domain.Meeting, offset *string) {
	if offset != nil {
		meeting.Session.GMTOffset = strings.Join(strings.Split(*offset, ":")[:2], "")
	}
}

func setSessionStartDate(meeting *domain.Meeting, start *string) {
	if start != nil {
		meeting.Session.StartDate, _ = time.Parse(f1APIDateLayout, *start+meeting.Session.GMTOffset)
	}
}

func setSessionEndDate(meeting *domain.Meeting, end *string) {
	if end != nil {
		meeting.Session.EndDate, _ = time.Parse(f1APIDateLayout, *end+meeting.Session.GMTOffset)
	}
}

func setSessionType(meeting *domain.Meeting, t *string) {
	if t != nil {
		switch *t {
		case "Test":
			meeting.Session.Type = domain.SessionTypeTest
		case "Practice":
			meeting.Session.Type = domain.SessionTypePractice
		case "Qualifying":
			meeting.Session.Type = domain.SessionTypeQualifying
		case "Race":
			meeting.Session.Type = domain.SessionTypeRace
		case "Unknown":
			meeting.Session.Type = domain.SessionTypeUnknown
		}
	}
}

func setSessionStatus(meeting *domain.Meeting, s *string) {
	if s != nil {
		switch *s {
		case "Started":
			meeting.Session.Status = domain.SessionStatusStarted
		case "Ended":
			meeting.Session.Status = domain.SessionStatusEnded
		case "Finished":
			meeting.Session.Status = domain.SessionStatusEnded
		}
	}
}

func setSessionPart(meeting *domain.Meeting, part *int) {
	if part != nil {
		meeting.Session.Part = *part
	}
}

/* Private types
------------------------------------------------------------------------------------------------- */

// negotiateResponse represents the response body of the F1 Live Timing negotiate API.
type negotiateResponse struct {
	Url                     string  `json:"Url"`
	ConnectionToken         string  `json:"ConnectionToken"`
	ConnectionId            string  `json:"ConnectionId"`
	KeepAliveTimeout        float64 `json:"KeepAliveTimeout"`
	DisconnectTimeout       float64 `json:"DisconnectTimeout"`
	ConnectionTimeout       float64 `json:"ConnectionTimeout"`
	TryWebSockets           bool    `json:"TryWebSockets"`
	ProtocolVersion         string  `json:"ProtocolVersion"`
	TransportConnectTimeout float64 `json:"TransportConnectTimeout"`
	LongPollDelay           float64 `json:"LongPollDelay"`
}
