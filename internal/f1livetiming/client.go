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
		_, _, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				conn.Close(websocket.StatusNormalClosure, "client closed")
			} else {
				c.doneCh <- err
			}
			return
		}
		// No errors, process the message from the livetiming API
		// c.processMessage(msg)
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
