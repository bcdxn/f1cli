package f1livetiming

import (
	"encoding/json"
	"strconv"
)

type SessionDataEvent struct {
	Data ChangeSessionData
}

func (c *Client) writeReferenceToSessionDataChannel(m any) {
	if c.SessionDataChannel == nil {
		// The consumer did not ask for session data events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var referenceSessionData ReferenceSessionData
	err = json.Unmarshal(s, &referenceSessionData)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	sessionData := ChangeSessionData{
		Series:       make(map[string]SessionDataSeries),
		StatusSeries: make(map[string]SessionDataStatusSeries),
	}
	for i, data := range referenceSessionData.Series {
		sessionData.Series[strconv.Itoa(i)] = data
	}
	for i, data := range referenceSessionData.StatusSeries {
		sessionData.StatusSeries[strconv.Itoa(i)] = data
	}

	sd := SessionDataEvent{
		Data: sessionData,
	}

	c.SessionDataChannel <- sd
}

func (c *Client) writeChangeToSessionDataChannel(m any) {
	if c.SessionDataChannel == nil {
		// The consumer did not ask for session data events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var sessionData ChangeSessionData
	err = json.Unmarshal(s, &sessionData)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	sd := SessionDataEvent{
		Data: sessionData,
	}

	c.SessionDataChannel <- sd
}
