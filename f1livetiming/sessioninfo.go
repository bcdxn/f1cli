package f1livetiming

import (
	"encoding/json"
)

type SessionInfoEvent struct {
	Data SessionInfo
}

func (c *Client) writeToSessionInfoChannel(m any) {
	if c.SessionInfoChannel == nil {
		// The consumer did not ask for session info events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var sessionInfo SessionInfo
	err = json.Unmarshal(s, &sessionInfo)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	sie := SessionInfoEvent{
		Data: sessionInfo,
	}

	c.SessionInfoChannel <- sie
}
