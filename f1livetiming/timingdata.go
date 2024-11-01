package f1livetiming

import "encoding/json"

type TimingDataEvent struct {
	Data TimingData
}

func (c *Client) writeToTimingDataChannel(m any) {
	if c.TimingDataChannel == nil {
		// The consumer did not ask for timing data events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var timingData TimingData
	err = json.Unmarshal(s, &timingData)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	td := TimingDataEvent{
		Data: timingData,
	}

	c.TimingDataChannel <- td
}
