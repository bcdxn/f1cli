package f1livetiming

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	d := make(chan struct{})
	i := make(chan os.Signal, 1)
	c := NewClient(i, d)

	expected := "https://livetiming.formula1.com"
	if c.BaseURL != expected {
		t.Errorf("Client.Host was not defaulted to the correct value, expected '%s', found '%s'", expected, c.BaseURL)
	}

	c = NewClient(i, d, WithBaseURL("http://test.com"))
	expected = "http://test.com"
	if c.BaseURL != expected {
		t.Errorf("Client.Host was not set to the correct value, expected '%s', found '%s'", expected, c.BaseURL)
	}
}

func TestNegotiate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer ts.Close()

	d := make(chan struct{})
	i := make(chan os.Signal, 1)
	c := NewClient(i, d, WithBaseURL(ts.URL))

	c.Negotiate()

	e := "connection-token"
	if c.ConnectionToken != e {
		t.Errorf("Client.ConnectionToken expected '%s', found '%s'", e, c.ConnectionToken)
	}
}
