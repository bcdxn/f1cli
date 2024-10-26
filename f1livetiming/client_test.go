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

func TestReferenceMessage(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	sessionInfoCh := make(chan SessionInfoEvent)
	referenceDataMsg, err := os.ReadFile("./testdata/messages-with-full-reference.json")
	if err != nil {
		t.Error("unable to read static data required for test setup", err)
	}
	// start test server
	ts, _ := newWSTestServer(t, func() websocket.Handler {
		return func(conn *websocket.Conn) {
			defer conn.Close()
			var msg string
			websocket.Message.Receive(conn, &msg)
			// Send message
			websocket.Message.Send(conn, referenceDataMsg)
		}
	}())
	defer ts.Close()
	// create and connect client to server
	c := NewClient(
		i,
		d,
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWs(t, ts.URL)),
		WithSessionInfoChannel(sessionInfoCh),
	)
	c.Negotiate()
	go c.Connect()
	// process and test session info event
	for wait := true; wait; {
		select {
		case err := <-d:
			wait = false
			if err != nil {
				t.Errorf("should not have errored but found '%s'", err.Error())
			}
		case e := <-sessionInfoCh:
			if e.Data.Meeting.Name != "United States Grand Prix" {
				t.Errorf("incorrect Name - expected '%s' but found '%s", "United States Grand Prix", e.Data.Meeting.Name)
			}
			if e.Data.Type != "Race" {
				t.Errorf("incorrect session type - expected '%s' but found '%s", "Race", e.Data.Type)
			}
			close(i)
		}
	}
}

// TestF1LivingTimingMessages ensures that messages
func TestWeatherDataMessages(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	weatherCh := make(chan WeatherDataEvent)
	weatherMsg, err := os.ReadFile("./testdata/messages-with-weather.json")
	if err != nil {
		t.Error("unable to read static data required for test setup", err)
		return
	}
	// start test server
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
	// create and connect client to server
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
		case e := <-weatherCh:
			if e.Data.AirTemp != "28.5" {
				t.Errorf("incorrect AirTemp - expected '%s' but found '%s", "28.5", e.Data.AirTemp)
			}
			close(i)
		}
	}
}

func TestF1RaceControlMessages(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	racectrlCh := make(chan RaceControlEvent)
	racectrlMsg, err := os.ReadFile("./testdata/messages-with-racectrl.json")
	if err != nil {
		t.Error("unable to read static data required for test setup", err)
		return
	}
	// start test server
	ts, _ := newWSTestServer(t, func() websocket.Handler {
		return func(conn *websocket.Conn) {
			defer conn.Close()
			var msg string
			websocket.Message.Receive(conn, &msg)
			// Send message
			websocket.Message.Send(conn, racectrlMsg)
		}
	}())
	defer ts.Close()
	// create and connect client to server
	c := NewClient(
		i,
		d,
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWs(t, ts.URL)),
		WithRaceControlChannel(racectrlCh),
	)
	c.Negotiate()
	go c.Connect()
	// process and test race control event
	msgCount := 0
	for wait := true; wait; {
		select {
		case err := <-d:
			wait = false
			if err != nil {
				t.Errorf("should not have errored but found '%s'", err.Error())
			}
		case e := <-racectrlCh:
			msgCount++
			// check all 4 messages
			switch msgCount {
			case 1:
				if e.Data.Lap != 19 {
					t.Errorf("invalid race ctrl event - expected '%d' but found '%d'", 19, e.Data.Lap)
				}
				if e.Data.Category != "Flag" {
					t.Errorf("invalid race ctrl event - expected '%s' but found '%s'", "Flag", e.Data.Category)
				}
			case 2:
				if e.Data.Category != "Drs" {
					t.Errorf("invalid race ctrl event - expected '%s' but found '%s'", "Drs", e.Data.Category)
				}
				if e.Data.Status != "DISABLED" {
					t.Errorf("invalid race ctrl event - expected '%s' but found '%s'", "DISABLED", e.Data.Status)
				}
			case 4:
				close(i)
			}
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
