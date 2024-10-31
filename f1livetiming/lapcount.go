package f1livetiming

import "encoding/json"

type LapCountEvent struct {
	Data LapCount
}

func (c *Client) writeToLapCountChannel(m any) {
	if c.LapCountChannel == nil {
		// The consumer did not ask for lap count events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var lapCount LapCount
	err = json.Unmarshal(s, &lapCount)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	lc := LapCountEvent{
		Data: lapCount,
	}

	c.LapCountChannel <- lc
}
