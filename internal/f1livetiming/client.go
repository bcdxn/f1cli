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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/coder/websocket"
)

// New returns a new F1 LiveTiming API Client.
func New(opts ...ClientOption) Client {
	// create a default instance of the client
	c := Client{
		drivers:        make(map[string]domain.Driver),
		meeting:        domain.NewMeeting(),
		raceCtrlMsgs:   make([]domain.RaceCtrlMsg, 0),
		driversCh:      make(chan map[string]domain.Driver),
		meetingCh:      make(chan domain.Meeting),
		raceCtrlMsgsCh: make(chan []domain.RaceCtrlMsg),
		doneCh:         make(chan error),
		logger:         slog.Default(),
		httpBaseURL:    "http://localhost:3000",
		wsBaseURL:      "ws://localhost:3000",
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
	raceCtrlMsgs    []domain.RaceCtrlMsg
	connectionToken string
	cookie          string
	// channels
	driversCh      chan map[string]domain.Driver
	meetingCh      chan domain.Meeting
	raceCtrlMsgsCh chan []domain.RaceCtrlMsg
	doneCh         chan error
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
func (c Client) RaceCtrlMsgs() <-chan []domain.RaceCtrlMsg {
	return c.raceCtrlMsgsCh
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

var (
	// The F1 API returns a mixed-type map which makes ummarshalling to strongly typed structs
	// challenging, so we just strip the offending property from the message string using the kfRe
	// regular expression.
	kfRe = regexp.MustCompile(`,\s*"_kf":\s*(?:true|false)(,[^}])?`)
)

// processMessage checks the message coming the F1 LiveTiming Client to see if it is a 'change'
// message or a 'reference' message and handles them appropriately, transforming the message into
// 1 to none or many messages that can be written to the client channels.
func (c *Client) processMessage(msg []byte) {
	// Always try to parse a change message first since there is only 1 reference message and
	// tens of thousands of change messages over the course of a session
	var changeData f1ChangeMessage
	err := json.Unmarshal(msg, &changeData)
	if err == nil && len(changeData.ChangeSetId) > 0 && len(changeData.Messages) > 0 {
		c.logger.Debug("received change data message")
		c.processChangeMessage(changeData)
		return
	}
	// Next try to parse a reference data message
	referenceMsg := kfRe.ReplaceAllString(string(msg), "")
	var referenceData f1ReferenceMessage
	err = json.Unmarshal([]byte(referenceMsg), &referenceData)
	if err == nil && referenceData.MessageInterval != "" {
		c.logger.Debug("received reference data message")
		c.logger.Debug(string(msg))
		c.processReferenceMessage(referenceData)
		return
	}
	// The message wasn't a known 'change' or 'reference' message type
	c.logger.Debug("unhandled message", "msg", msg)
}

// processChangeMessage handles an incoming change message from the F1 Live Timing API; change
// messages represent deltas to the original reference message and all preceeding change messages.
// Once processed, a simplified event is emitted for use by the TUI.
func (c *Client) processChangeMessage(changeMessage f1ChangeMessage) {
	for _, m := range changeMessage.Messages {
		if m.Hub == "Streaming" && m.Message == "feed" && len(m.Arguments) == 3 {
			msgType := m.Arguments[0]
			msgData := m.Arguments[1]
			// Marshal the data part of the message so that we can unmarshal into strongly typed messages
			// based on the messageType value.
			msg, err := json.Marshal(msgData)
			if err != nil {
				c.logger.Warn("unable to marshal msg", "msg", msg)
				return
			}
			switch msgType {
			case "DriverList":
				c.updateDriverIntrinsicData(c.ummarshalDriverListMsg(msg))
			// case "TimingData":
			// 	c.handleDriverTimingData(msg)
			// case "SessionInfo":
			// 	c.handleSessionInfoMsg(msg)
			// case "SessionData":
			// 	c.handleSessionDataMsg(msg)
			// case "LapCount":
			// 	c.handleLapCountMsg(msg)
			// case "TimingAppData":
			// 	c.handleTimingAppDataMsg(msg)
			default:
				c.logger.Warn("unknown change message", "type", msgType, "msg", msg)
			}
		}
	}
}

func (c *Client) processReferenceMessage(referenceMessage f1ReferenceMessage) {
	c.updateSessionInfo(referenceMessage.Reference.SessionInfo)
	c.updateSessionData(
		changeSessionDateFromReference(referenceMessage.Reference.SessionData),
	)
	c.updateDriverIntrinsicData(referenceMessage.Reference.DriverList)
}

/* Message Unmarshalers
------------------------------------------------------------------------------------------------- */

const (
	f1APIDateLayout = "2006-01-02T15:04:05-0700" // date format used by the F1 LiveTiming API
)

// unmarshalSessionInfo converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalSessionInfo(msg []byte) sessionInfo {
	var s sessionInfo
	err := json.Unmarshal(msg, &s)
	if err != nil {
		c.logger.Warn("session info msg in unknown format", "msg", string(msg))
	}

	return s
}

// unmarshalSessionData converts the websocket message to a strongly typed struct.
func (c *Client) unmarshalSessionDataMsg(msg []byte) changeSessionData {
	var s changeSessionData
	err := json.Unmarshal(msg, &s)
	if err != nil {
		c.logger.Warn("session data msg in unknown format", "msg", string(msg))
	}

	return s
}

// ummarshalDriverListMsg converts the websocket message to a strongly typed map of structs.
func (c *Client) ummarshalDriverListMsg(msg []byte) map[string]driverData {
	var d map[string]driverData
	err := json.Unmarshal(msg, &d)
	if err != nil {
		c.logger.Warn("driver data msg in unknown format", "msg", string(msg))
	}
	return d
}

/* Channel Updaters
------------------------------------------------------------------------------------------------- */

// updateSessionInfo converts a SessionInfo msg from the F1 Live Timing API to the `Meeting` and
// `Session` domain models  stored in the client's internal state store and writes the full state
// of the meeting for consumers to read.
func (c *Client) updateSessionInfo(session sessionInfo) {
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
	c.meetingCh <- c.meeting
}

// updateSessionData converts a SessionData msg from the F1 LiveTiming API to the `Session` domain
// model and writes the full state of the meeting/session for consumers to read.
func (c *Client) updateSessionData(session changeSessionData) {
	seriesKeys := make([]string, 0)
	for key := range session.Series {
		seriesKeys = append(seriesKeys, key)
	}
	// Access the series messages in order so that we end up on the latest entry
	sort.Strings(seriesKeys)
	for _, key := range seriesKeys {
		setSessionPart(&c.meeting, session.Series[key].QualifyingPart)
	}

	c.meetingCh <- c.meeting
}

// updateDriverIntrinsicData converts DriverList msg from the F1 Live Timing API to the Driver
// domain models stored in the client's internal state store and writes the full drivers store to
// the drivers channel for consumers to read.
func (c *Client) updateDriverIntrinsicData(driverDataMsg map[string]driverData) {
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
	c.driversCh <- c.drivers
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
		meeting.Session.Type = domain.SessionType(*t)
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
