package f1livetiming

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/websocket"
)

type Client struct {
	Interrupt         chan os.Signal
	Done              chan struct{}
	WeatherDataEvents chan WeatherDataEvent
	ConnectionToken   string
	Cookie            string
	BaseURL           string
}

// NewClient creates and returns a new F1 LiveTiming Client for retrieving real-time data from
// active F1 sessions.
func NewClient(
	// Client will gracefully close websocket when interrupt signal is received
	interrupt chan os.Signal,
	// Client will signal to parents that the websocket has been closed; parents should wait for this
	// signal before closing
	done chan struct{},
	opts ...ClientOption,
) *Client {
	c := &Client{
		Interrupt: interrupt,
		Done:      done,
		BaseURL:   "https://livetiming.formula1.com",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type ClientOption = func(c *Client)

func WithBaseURL(baseUrl string) ClientOption {
	return func(c *Client) {
		c.BaseURL = baseUrl
	}
}

func WithWeatherEvents(weatherEvents chan WeatherDataEvent) ClientOption {
	return func(c *Client) {
		c.WeatherDataEvents = weatherEvents
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
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid BaseURL: %w", err)
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
		return fmt.Errorf("error negotiating f1 livetiming api connection: %w", err)
	}
}

func (c *Client) Connect() {
	<-c.Interrupt
	fmt.Println("interrupt received")
	close(c.Done)
}

func (c *Client) getWebsocketConfig() (*websocket.Config, error) {
	var config *websocket.Config
	b, err := url.Parse(c.BaseURL)
	if err != nil {
		return config, fmt.Errorf("invalid BaseURL: %w", err)
	}

	u := url.URL{
		Scheme: "wss",
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

func newWeatherDataEvent(d WeatherData) WeatherDataEvent {
	return WeatherDataEvent{
		Name: "WeatherData",
		Data: d,
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
