package f1livetiming

import "encoding/json"

type RaceControlEvent struct {
	Data RaceControlMessage
}

func (c *Client) writeReferenceToRaceControlChannel(m any) {
	if c.RaceControlChannel == nil {
		// The consumer did not ask for race control events; no need to process
		return
	}
	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var msgs RaceControlMessages
	err = json.Unmarshal(s, &msgs)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	for _, msg := range msgs.Messages {
		rce := RaceControlEvent{
			Data: msg,
		}

		c.RaceControlChannel <- rce
	}
}

func (c *Client) writeChangeToRaceControlChannel(m any) {
	if c.RaceControlChannel == nil {
		// The consumer did not ask for race control events; no need to process
		return
	}
	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var msgs RaceControlMessagesMap
	err = json.Unmarshal(s, &msgs)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	for _, msg := range msgs.Messages {
		rce := RaceControlEvent{
			Data: msg,
		}

		c.RaceControlChannel <- rce
	}
}
