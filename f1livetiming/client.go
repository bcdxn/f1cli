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

type Client struct {
	Interrupt       chan struct{}
	Done            chan error
	WeatherChannel  chan WeatherDataEvent
	ConnectionToken string
	Cookie          string
	HTTPBaseURL     string
	WSBaseURL       string
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
		Interrupt:   interruptChannel,
		Done:        doneChannel,
		HTTPBaseURL: "https://livetiming.formula1.com",
		WSBaseURL:   "https://livetiming.formula1.com",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

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

func WithWeatherChannel(weatherEvents chan WeatherDataEvent) ClientOption {
	return func(c *Client) {
		c.WeatherChannel = weatherEvents
	}
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

func (c *Client) Negotiate() error {
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
		return nil
	default:
		return fmt.Errorf("error negotiating f1 livetiming api connection: %w", errors.New(resp.Status))
	}
}

func (c *Client) Connect() {
	if c.ConnectionToken == "" {
		c.Done <- errors.New("client.Negotiate() was not called or was unnsuccessful")
		close(c.Done)
		return
	}

	config, err := c.getWebsocketConfig()
	if err != err {
		c.Done <- err
		close(c.Done)
		return
	}

	sock, err := websocket.DialConfig(config)
	if err != nil {
		c.Done <- fmt.Errorf("error establishing websocket connection: %w", err)
		close(c.Done)
		return
	}
	defer sock.Close()

	fmt.Println("we here!")

	sendSubscribe(sock)

	listening := true
	go func() {
		for listening {
			var msg string
			err = websocket.Message.Receive(sock, &msg)
			if err != nil && err.Error() == "EOF" {
				return
			} else if err != nil {
				return
			}
			c.processMessage(msg)
		}
	}()
	<-c.Interrupt // wait on interrupt
	listening = false
	c.Done <- err // write any errors to the done channel before closing
	close(c.Done) // close client done channel so other's know we've closed the connection gracefully
}

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

func sendSubscribe(sock *websocket.Conn) {
	websocket.Message.Send(sock, `
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

type SignalrMessage struct {
	Hub       string     `json:"H"`
	Method    string     `json:"M"`
	Arguments [][]string `json:"A"`
	Interval  uint8      `json:"I"`
}

type F1Message struct {
	Messages []struct {
		Hub       string `json:"H"`
		Message   string `json:"M"`
		Arguments []any  `json:"A"`
	} `json:"M"`
}

func (c *Client) processMessage(msg string) {
	var messageData F1Message
	err := json.Unmarshal([]byte(msg), &messageData)
	if err != nil {
		fmt.Println("ERROR UNMARSHALLING MSG:", msg)
		return
	}

	for _, m := range messageData.Messages {
		if m.Hub == "Streaming" && m.Message == "feed" && len(m.Arguments) == 3 {
			switch m.Arguments[0] {
			case "WeatherData":
				c.WeatherChannel <- newWeatherDataEvent(m.Arguments)
			}
		}
	}
}

type WeatherData struct {
	AirTemp       string `json:"AirTemp"`
	Humidity      string `json:"Humidity"`
	Pressure      string `json:"Pressure"`
	Rainfall      string `json:"Rainfall"`
	TrackTemp     string `json:"TrackTemp"`
	WindDirection string `json:"WindDirection"`
	WindSpeed     string `json:"WindSpeed"`
}

type WeatherDataEvent struct {
	Name string
	Data WeatherData
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

func newWeatherDataEvent(args []any) WeatherDataEvent {
	wde := WeatherDataEvent{
		Name: "WeatherData",
		Data: WeatherData{},
	}

	if strMap, ok := args[1].(map[string]any); ok {
		if str, ok := strMap["AirTemp"].(string); ok {
			wde.Data.AirTemp = str
		}
		if str, ok := strMap["Humidity"].(string); ok {
			wde.Data.Humidity = str
		}
		if str, ok := strMap["Pressure"].(string); ok {
			wde.Data.Pressure = str
		}
		if str, ok := strMap["Rainfall"].(string); ok {
			wde.Data.Rainfall = str
		}
		if str, ok := strMap["TrackTemp"].(string); ok {
			wde.Data.TrackTemp = str
		}
		if str, ok := strMap["WindDirection"].(string); ok {
			wde.Data.WindDirection = str
		}
		if str, ok := strMap["WindSpeed"].(string); ok {
			wde.Data.WindSpeed = str
		}
	}

	return wde
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
