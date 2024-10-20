package f1livetiming

import (
	"fmt"
	"os"
)

type Client struct {
	Interrupt         chan os.Signal
	Done              chan bool
	WeatherDataEvents chan WeatherDataEvent
	Negotiation       NegotiateResponse
}

func NewClient(
	// Client will gracefully close websocket when interrupt signal is received
	interrupt chan os.Signal,
	// Client will signal to parents that the websocket has been closed; parents should wait for this
	// signal before closing
	done chan bool,
	// Client will write weather data events to this channel for
	weatherEvents chan WeatherDataEvent,
) *Client {
	return &Client{
		Interrupt:         interrupt,
		Done:              done,
		WeatherDataEvents: weatherEvents,
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

func (c *Client) Negotiate() {

}

func (c *Client) Connect() {
	<-c.Interrupt
	fmt.Println("interrupt received")
	c.Done <- true
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
