package f1livetiming

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"golang.org/x/net/websocket"
)

func TestNewClient(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	c := NewClient(i, d)

	expected := "https://livetiming.formula1.com"
	if c.HTTPBaseURL != expected {
		t.Errorf("Client.HTTPBaseURL was not defaulted to the correct value, expected '%s', found '%s'", expected, c.HTTPBaseURL)
	}
	if c.WSBaseURL != expected {
		t.Errorf("Client.WSBaseURL was not defaulted to the correct value, expected '%s', found '%s'", expected, c.WSBaseURL)
	}

	h := "http://test.com"
	w := httpToWs(t, "http://test.com")
	c = NewClient(i, d, WithHTTPBaseURL(h), WithWSBaseURL(w))
	if c.HTTPBaseURL != h {
		t.Errorf("Client.HTTPBaseURL was not set to the correct value, expected '%s', found '%s'", h, c.HTTPBaseURL)
	}
	if c.WSBaseURL != w {
		t.Errorf("Client.HTTPBaseURL was not set to the correct value, expected '%s', found '%s'", w, c.WSBaseURL)
	}
}

func TestNegotiate(t *testing.T) {
	ts := newHttpTestServer(t)
	defer ts.Close()

	i := make(chan struct{})
	d := make(chan error)
	c := NewClient(i, d, WithHTTPBaseURL(ts.URL))

	c.Negotiate()

	e := "connection-token"
	if c.ConnectionToken != e {
		t.Errorf("Client.ConnectionToken expected '%s', found '%s'", e, c.ConnectionToken)
	}
}

// TestConnectWait ensures that the caller is properly notified when the client shuts down
func TestConnectWithoutNegotiate(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	c := NewClient(i, d)

	go c.Connect()

	err := <-d
	e := "client.Negotiate() was not called or was unnsuccessful"
	if err == nil || err.Error() != e {
		t.Errorf("Client.Connect() should require successful Client.Negotiate call but got err: '%s'", err.Error())
	}
}

// TestConnectWait ensures that the caller is properly notified when the client shuts down
func TestConnectSubscribe(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)

	ts, _ := newWSTestServer(t, func() websocket.Handler {
		return func(conn *websocket.Conn) {
			defer conn.Close()
			var msg string
			websocket.Message.Receive(conn, &msg)

			re := regexp.MustCompile(`"M": "Subscribe",`)
			if !re.MatchString(msg) {
				t.Errorf("expected first message to be a Subscribe message, but found %s", msg)
			}
			close(i)
		}
	}())
	defer ts.Close()

	c := NewClient(i, d, WithHTTPBaseURL(ts.URL), WithWSBaseURL(httpToWs(t, ts.URL)))

	c.Negotiate()
	go c.Connect()

	err := <-d
	if err != nil {
		t.Errorf("did not expect error but found: '%s'", err.Error())
	}
}

// TestF1LivingTimingMessages ensures that messages
func TestF1LivingTimingMessages(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	weatherCh := make(chan WeatherDataEvent)
	weatherMsg, err := os.ReadFile("./testdata/message.json")
	if err != nil {
		t.Error("unable to read static data required for test setup", err)
	}

	ts, _ := newWSTestServer(t, func() websocket.Handler {
		return func(conn *websocket.Conn) {
			defer conn.Close()
			var msg string
			websocket.Message.Receive(conn, &msg)
			// Send message
			websocket.Message.Send(conn, weatherMsg)
		}
	}())
	defer ts.Close()

	c := NewClient(
		i,
		d,
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWs(t, ts.URL)),
		WithWeatherChannel(weatherCh),
	)

	c.Negotiate()
	go c.Connect()

	// process and test weather event
	for wait := true; wait; {
		select {
		case err := <-d:
			wait = false
			if err != nil {
				t.Errorf("should not have errored but found '%s'", err.Error())
			}
		case we := <-weatherCh:
			if we.Name != "WeatherData" {
				t.Errorf("invalid weather event - expected name '%s' but found '%s'", "WeatherData", we.Name)
			}
			if we.Data.AirTemp != "28.5" {
				t.Errorf("incorrect AirTemp - expected '%s' but found '%s", "28.5", we.Data.AirTemp)
			}
			close(i)
		}
	}
}

func newHttpTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cookie", "test-cookie")
		fmt.Fprintln(w, `
		{
			"Url": "/signalr",
			"ConnectionToken": "connection-token",
			"ConnectionId": "connection-id",
			"KeepAliveTimeout": 20.0,
			"DisconnectTimeout": 30.0,
			"ConnectionTimeout": 110.0,
			"TryWebSockets": true,
			"ProtocolVersion": "1.5",
			"TransportConnectTimeout": 10.0,
			"LongPollDelay": 1.0
		}
		`)
	})

	return httptest.NewServer(mux)
}

func newWSTestServer(t *testing.T, h websocket.Handler) (*httptest.Server, *websocket.Server) {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cookie", "test-cookie")
		fmt.Fprintln(w, `
		{
			"Url": "/signalr",
			"ConnectionToken": "connection-token",
			"ConnectionId": "connection-id",
			"KeepAliveTimeout": 20.0,
			"DisconnectTimeout": 30.0,
			"ConnectionTimeout": 110.0,
			"TryWebSockets": true,
			"ProtocolVersion": "1.5",
			"TransportConnectTimeout": 10.0,
			"LongPollDelay": 1.0
		}
		`)
	})

	var ws websocket.Server
	mux.HandleFunc("/signalr/connect", func(w http.ResponseWriter, r *http.Request) {
		ws = websocket.Server{
			Handler: h,
		}
		ws.ServeHTTP(w, r)
	})
	s := httptest.NewServer(mux)

	return s, &ws
}

func httpToWs(t *testing.T, u string) string {
	t.Helper()
	httpsRe := regexp.MustCompile("https://")
	httpRe := regexp.MustCompile("http://")

	wsUrl := httpsRe.ReplaceAllString(u, "wss://")
	return httpRe.ReplaceAllString(wsUrl, "ws://")
}

// func sendMessage(t *testing.T, ws *websocket.Conn, msg string) {
// 	t.Helper()
// 	ws.Write([]byte(msg))
// }
