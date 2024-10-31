package f1livetiming

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/websocket"
)

type testLogger struct{}

func (l testLogger) Debug(string, ...any) {}
func (l testLogger) Error(string, ...any) {}

func TestNewClient(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	c := NewClient(i, d, WithLogger(testLogger{}))

	expected := "https://livetiming.formula1.com"
	if c.HTTPBaseURL != expected {
		t.Errorf("Client.HTTPBaseURL was not defaulted to the correct value, expected '%s', found '%s'", expected, c.HTTPBaseURL)
	}
	if c.WSBaseURL != expected {
		t.Errorf("Client.WSBaseURL was not defaulted to the correct value, expected '%s', found '%s'", expected, c.WSBaseURL)
	}

	h := "http://test.com"
	w := httpToWs(t, "http://test.com")
	c = NewClient(i, d, WithHTTPBaseURL(h), WithWSBaseURL(w), WithLogger(testLogger{}))
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
	c := NewClient(i, d, WithHTTPBaseURL(ts.URL), WithLogger(testLogger{}))

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
	c := NewClient(i, d, WithLogger(testLogger{}))

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

	c := NewClient(i, d, WithHTTPBaseURL(ts.URL), WithWSBaseURL(httpToWs(t, ts.URL)), WithLogger(testLogger{}))

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
	driverListCh := make(chan DriverListEvent)
	referenceDataMsg, err := os.ReadFile("./testdata/reference-message.json")
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
		WithDriverListChannel(driverListCh),
		WithSessionInfoChannel(sessionInfoCh),
		WithLogger(testLogger{}),
	)
	c.Negotiate()
	go c.Connect()
	// process and test session info event
	msgCount := 0
	for listening := true; listening; {
		select {
		case err := <-d:
			listening = false
			if err != nil {
				t.Errorf("should not have errored but found '%s'", err.Error())
			}
		case e := <-sessionInfoCh:
			msgCount++
			testSessionInfo(t, e)
		case e := <-driverListCh:
			msgCount++
			testDriverList(t, e)
		}
		// Interrupt the client if we've processed all of the messages we need to process
		if msgCount >= 2 && listening {
			close(i)
		}
	}
}

func TestChangeMessages(t *testing.T) {
	i := make(chan struct{})
	d := make(chan error)
	driverListCh := make(chan DriverListEvent)
	lapCountCh := make(chan LapCountEvent)
	timingDataCh := make(chan TimingDataEvent)
	sessionInfoCh := make(chan SessionInfoEvent)
	racectrlMsgCh := make(chan RaceControlEvent)
	weatherDataCh := make(chan WeatherDataEvent)
	msgs := getAllMessages(t)

	// start test server
	ts, _ := newWSTestServer(t, func() websocket.Handler {
		return func(conn *websocket.Conn) {
			defer conn.Close()
			var msg string
			websocket.Message.Receive(conn, &msg)

			for _, msg := range msgs {
				// Send message
				websocket.Message.Send(conn, msg)
			}
		}
	}())
	defer ts.Close()

	// create and connect client to server
	c := NewClient(
		i,
		d,
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWs(t, ts.URL)),
		WithDriverListChannel(driverListCh),
		WithLapCountChannel(lapCountCh),
		WithTimingDataChannel(timingDataCh),
		WithSessionInfoChannel(sessionInfoCh),
		WithRaceControlChannel(racectrlMsgCh),
		WithWeatherChannel(weatherDataCh),
		WithLogger(testLogger{}),
	)
	c.Negotiate()
	go c.Connect()
	msgCount := 0

	// process and test events
	for listening := true; listening; {
		select {
		case err := <-d:
			listening = false
			if err != nil {
				t.Fatalf("should not have errored but found '%s'", err.Error())
			}
		case e := <-driverListCh:
			msgCount++
			testDriverList(t, e)
		case e := <-lapCountCh:
			msgCount++
			testLapCount(t, e)
		case e := <-timingDataCh:
			msgCount++
			testTimingData(t, e)
		case e := <-sessionInfoCh:
			msgCount++
			testSessionInfo(t, e)
		case e := <-racectrlMsgCh:
			msgCount++
			testRaceControlMessages(t, e)
		case e := <-weatherDataCh:
			msgCount++
			testWeatherData(t, e)
		}
		// Interrupt the client if we've processed all of the messages we need to process
		if msgCount >= len(msgs) && listening {
			close(i)
		}
	}
}

func testLapCount(t *testing.T, e LapCountEvent) {
	t.Run("LapCount", func(t *testing.T) {
		if e.Data.CurrentLap != 2 {
			t.Errorf("incorrect CurrentLap - expected '%d' but found '%d", 2, e.Data.CurrentLap)
		}
	})
}

func testTimingData(t *testing.T, e TimingDataEvent) {
	t.Run("TimingData", func(t *testing.T) {
		if driverData, ok := e.Data.Lines["55"]; ok {
			if driverData.GapToLeader != "+12.562" {
				t.Errorf("incorrect TimingData - expected '%s' but found '%s", "+12.562", driverData.GapToLeader)
			}
		} else {
			t.Error("timing data did not container expected driver")
		}
	})
}

func testDriverList(t *testing.T, e DriverListEvent) {
	t.Run("DriverList", func(t *testing.T) {
		if e.Data["4"].FirstName != "Lando" {
			t.Errorf("invalid driverlist event - expected '%s' but found '%s'", "Lando", e.Data["4"].FirstName)
		}
		if e.Data["4"].Line != 1 {
			t.Errorf("invalid driverlist event - expected '%d' but found '%d'", 1, e.Data["4"].Line)
		}
		if e.Data["4"].TeamColour != "FF8000" {
			t.Errorf("invalid driverlist event - expected '%s' but found '%s'", "FF8000", e.Data["4"].TeamColour)
		}
	})
}

func testSessionInfo(t *testing.T, e SessionInfoEvent) {
	t.Run("SessionInfo", func(t *testing.T) {
		if e.Data.Meeting.Name != "United States Grand Prix" {
			t.Errorf("incorrect Name - expected '%s' but found '%s", "United States Grand Prix", e.Data.Meeting.Name)
		}
		if e.Data.Type != "Race" {
			t.Errorf("incorrect session type - expected '%s' but found '%s", "Race", e.Data.Type)
		}
	})
}

func testRaceControlMessages(t *testing.T, e RaceControlEvent) {
	t.Run("RaceControlMessages", func(t *testing.T) {
		if e.Data.Lap != 19 {
			t.Errorf("invalid race ctrl event - expected '%d' but found '%d'", 19, e.Data.Lap)
		}
		if e.Data.Category != "Flag" {
			t.Errorf("invalid race ctrl event - expected '%s' but found '%s'", "Flag", e.Data.Category)
		}
	})
}

func testWeatherData(t *testing.T, e WeatherDataEvent) {
	if e.Data.AirTemp != "28.5" {
		t.Errorf("incorrect AirTemp - expected '%s' but found '%s", "28.5", e.Data.AirTemp)
	}
}

func getAllMessages(t *testing.T) []string {
	t.Helper()
	dir := strings.Join([]string{"testdata", "change-messages"}, string(os.PathSeparator))
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("error reading message files required for tests")
	}

	msgs := make([]string, 0)
	for _, file := range files {
		p := strings.Join([]string{dir, file.Name()}, string(os.PathSeparator))
		msg, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("error reading message file '%s' required for tests", p)
		}
		msgs = append(msgs, string(msg))
	}

	return msgs
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
