package f1livetiming

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/net/websocket"
)

/* Core Client API
------------------------------------------------------------------------------------------------- */

type Client struct {
	logger             Logger
	Interrupt          chan struct{}
	Done               chan error
	SessionInfoChannel chan SessionInfoEvent
	WeatherChannel     chan WeatherDataEvent
	RaceControlChannel chan RaceControlEvent
	DriverListChannel  chan DriverListEvent
	LapCountChannel    chan LapCountEvent
	TmingDataChannel   chan TimingDataEvent
	ConnectionToken    string
	Cookie             string
	HTTPBaseURL        string
	WSBaseURL          string
}

// NewClient creates and returns a new F1 LiveTiming Client for retrieving real-time data from
// active F1 sessions.
func NewClient(
	// Client will gracefully close websocket when interrupt signal is received
	interruptChannel chan struct{},
	// Client will signal to callers that the websocket is closed on this channel. It may also contain
	// errors
	doneChannel chan error,
	opts ...ClientOption,
) *Client {
	c := &Client{
		logger:      logger{},
		Interrupt:   interruptChannel,
		Done:        doneChannel,
		HTTPBaseURL: "https://livetiming.formula1.com",
		WSBaseURL:   "wss://livetiming.formula1.com",
	}

	for _, opt := range opts {
		opt(c)
	}

	c.logger.Debug("creating new F1 LiveTiming Client")

	return c
}

type NegotiateResponse struct {
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

// Negotiate calls the F1 Livetiming Signalr API, retreiving information required to start the
// websocket connection using the Connect function.
func (c *Client) Negotiate() error {
	c.logger.Debug("negotiating connection")
	u, err := url.Parse(c.HTTPBaseURL)
	if err != nil {
		return fmt.Errorf("invalid HTTPBaseURL: %w", err)
	}

	resp, err := http.DefaultClient.Do(&http.Request{
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
	})
	if err != nil {
		return fmt.Errorf("error sending f1 livetiming api negotiation request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		ct, err := parseNegotationConnectionToken(resp.Body)
		if err != nil {
			return fmt.Errorf("error parsing connection token: %w", err)
		}
		c.ConnectionToken = ct
		c.Cookie = resp.Header.Get("set-cookie")
		c.logger.Debug("successfully negotiated connection; connection token len:", len(ct))
		return nil
	default:
		return fmt.Errorf("error negotiating f1 livetiming api connection: %w", errors.New(resp.Status))
	}
}

// Connect opens a websocket connection and automatically sends the "Subscribe" method to the
// F1 Livetiming Signalr Server.
func (c *Client) Connect() {
	c.logger.Debug("connecting to signalr server for realtime updates")
	if c.ConnectionToken == "" {
		c.Done <- errors.New("client.Negotiate() was not called or was unnsuccessful")
		close(c.Done)
		return
	}

	config, err := c.getWebsocketConfig()
	if err != err {
		c.logger.Error("error establishing websocket connection:", err)
		c.Done <- err
		close(c.Done)
		return
	}
	c.logger.Debug("successfully configured signalr websocket")

	sock, err := websocket.DialConfig(config)
	if err != nil {
		c.logger.Error("error establishing websocket connection:", err)
		c.Done <- fmt.Errorf("error establishing websocket connection: %w", err)
		close(c.Done)
		return
	}
	defer sock.Close()
	c.logger.Debug("successfully connected to signalr websocket")

	err = sendSubscribe(sock)
	if err != nil {
		c.logger.Error("error establishing websocket connection:", err)
		c.Done <- fmt.Errorf("error establishing websocket connection: %w", err)
		close(c.Done)
		return
	}

	listening := true
	go func() {
		c.logger.Debug("listening on signalr websocket")
		for listening {
			var msg string
			err = websocket.Message.Receive(sock, &msg)
			if err != nil && err.Error() == "EOF" {
				err = nil // we can ignore this error; it just means the server closed
				c.logger.Debug("received EOF message livetiming API")
				return
			} else if err != nil {
				c.logger.Error("received error from livetiming API", err)
				return
			}
			c.logger.Debug("received f1 livetiming API message", msg)
			// Always try to parse a change message first since there is only 1 reference message and
			// tens of thousands of change messages over the course of a session
			var changeData F1ChangeMessage
			err := json.Unmarshal([]byte(msg), &changeData)
			if err == nil && len(changeData.ChangeSetId) > 0 && len(changeData.Messages) > 0 {
				c.logger.Debug("received change data message")
				c.processChangeMessage(changeData)
				continue
			}
			// Next try to parse a reference data message
			var referenceData F1ReferenceMessage
			err = json.Unmarshal([]byte(msg), &referenceData)
			if err == nil && referenceData.MessageInterval != "" {
				c.logger.Debug("received reference data message")
				c.processReferenceMessage(referenceData)
			}
			c.logger.Debug("done processing message")
		}
	}()

	c.logger.Debug("f1 client waiting for interrupt")
	<-c.Interrupt // wait on interrupt
	c.logger.Debug("f1 client received interrupt")
	listening = false
	c.logger.Debug("writing any error to done channel")
	c.Done <- err // write any errors to the done channel before closing
	c.logger.Debug("closing done channel")
	close(c.Done) // close client done channel so other's know we've closed the connection gracefully
}

/* Optional Function Parameters
------------------------------------------------------------------------------------------------- */

type ClientOption = func(c *Client)

func WithHTTPBaseURL(baseUrl string) ClientOption {
	return func(c *Client) {
		c.HTTPBaseURL = baseUrl
	}
}

func WithWSBaseURL(baseUrl string) ClientOption {
	return func(c *Client) {
		c.WSBaseURL = baseUrl
	}
}

func WithSessionInfoChannel(sessionInfoEvents chan SessionInfoEvent) ClientOption {
	return func(c *Client) {
		c.SessionInfoChannel = sessionInfoEvents
	}
}

func WithWeatherChannel(weatherEvents chan WeatherDataEvent) ClientOption {
	return func(c *Client) {
		c.WeatherChannel = weatherEvents
	}
}

func WithRaceControlChannel(raceCtrlEvents chan RaceControlEvent) ClientOption {
	return func(c *Client) {
		c.RaceControlChannel = raceCtrlEvents
	}
}

func WithDriverListChannel(driverlistEvents chan DriverListEvent) ClientOption {
	return func(c *Client) {
		c.DriverListChannel = driverlistEvents
	}
}

func WithLapCountChannel(lapCountEvents chan LapCountEvent) ClientOption {
	return func(c *Client) {
		c.LapCountChannel = lapCountEvents
	}
}

func WithTimingDataChannel(timingDataEvents chan TimingDataEvent) ClientOption {
	return func(c *Client) {
		c.TmingDataChannel = timingDataEvents
	}
}

func WithLogger(l Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

/* F1 Livetiming API Raw Message Types
------------------------------------------------------------------------------------------------- */

type SignalrMessage struct {
	Hub       string     `json:"H"`
	Method    string     `json:"M"`
	Arguments [][]string `json:"A"`
	Interval  uint8      `json:"I"`
}

type F1ChangeMessage struct {
	ChangeSetId string            `json:"C"`
	Messages    []F1NestedMessage `json:"M"`
}

type F1NestedMessage struct {
	Hub       string `json:"H"`
	Message   string `json:"M"`
	Arguments []any  `json:"A"`
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

func (c *Client) getWebsocketConfig() (*websocket.Config, error) {
	var config *websocket.Config
	b, err := url.Parse(c.WSBaseURL)
	if err != nil {
		return config, fmt.Errorf("invalid BaseURL: %w", err)
	}

	u := url.URL{
		Scheme: b.Scheme,
		Host:   b.Host,
		Path:   "/signalr/connect",
		RawQuery: url.Values{
			"connectionData":  {`[{"Name":"Streaming"}]`},
			"connectionToken": {c.ConnectionToken},
			"clientProtocol":  {"1.5"},
			"transport":       {"webSockets"},
		}.Encode(),
	}

	config, err = websocket.NewConfig(u.String(), u.String())
	if err != nil {
		return config, fmt.Errorf("error creating websocket config: %w", err)
	}

	config.Header = http.Header{
		"User-Agent":      {"BestHTTP"},
		"Accept-Encoding": {"gzip,identity"},
		"Cookie":          {c.Cookie},
	}

	return config, nil
}

func sendSubscribe(sock *websocket.Conn) error {
	return websocket.Message.Send(sock, `
		{
			"H": "Streaming",
			"M": "Subscribe",
			"A": [[
				"Heartbeat",
				"CarData.z",
				"Position.z",
				"ExtrapolatedClock",
				"TopThree",
				"RcmSeries",
				"TimingStats",
				"TimingAppData",
				"WeatherData",
				"TrackStatus",
				"DriverList",
				"RaceControlMessages",
				"SessionInfo",
				"SessionData",
				"LapCount",
				"TimingData"
			]],
			"I": 5
		}
	`)
}

func (c *Client) processReferenceMessage(referenceMessage F1ReferenceMessage) {
	c.writeToSessionInfoChannel(referenceMessage.Reference.SessionInfo)
	c.writeToWeatherChannel(referenceMessage.Reference.WeatherData)
	c.writeToDriverListChannel(referenceMessage.Reference.DriverList)
	c.writeToLapCountChannel(referenceMessage.Reference.LapCount)
	c.writeToTimingDataChannel(referenceMessage.Reference.TimingData)
}

func (c *Client) processChangeMessage(changeMessage F1ChangeMessage) {
	for _, m := range changeMessage.Messages {
		if m.Hub == "Streaming" && m.Message == "feed" && len(m.Arguments) == 3 {
			msgType := m.Arguments[0]
			msgData := m.Arguments[1]
			switch msgType {
			case "WeatherData":
				c.writeToWeatherChannel(msgData)
			case "RaceControlMessages":
				c.writeToRaceControlChannel(msgData)
			case "SessionInfo":
				c.writeToSessionInfoChannel(msgData)
			case "DriverList":
				c.writeToDriverListChannel(msgData)
			case "LapCount":
				c.writeToLapCountChannel(msgData)
			case "TimingData":
				c.writeToTimingDataChannel(msgData)
			default:
				c.logger.Debug("unknown change message type:", msgData)
			}
		}
	}
}

func parseNegotationConnectionToken(body io.ReadCloser) (string, error) {
	var n NegotiateResponse
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

/* Inner logging implementation
------------------------------------------------------------------------------------------------- */

type Logger interface {
	Debug(msg string, things ...any)
	Error(msg string, things ...any)
}

type logger struct{}

func (l logger) Debug(msg string, things ...any) {
	line := append([]any{msg}, things)
	fmt.Println(line...)
}

func (l logger) Error(msg string, things ...any) {
	line := append([]any{"ERROR:", msg}, things)
	fmt.Println(line...)
}
