package f1livetiming

import (
	"encoding/json"
	"strconv"
)

type TimingAppDataEvent struct {
	Data map[string]DriverTimingAppData
}

func (c *Client) writeReferenceToTimingAppDataChannel(m any) {
	if c.TimingAppDataChannel == nil {
		// The consumer did not ask for timing data events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var referenceTimingAppData ReferenceTimingAppData
	err = json.Unmarshal(s, &referenceTimingAppData)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	timingAppData := make(map[string]DriverTimingAppData)

	for key, data := range referenceTimingAppData.Lines {
		stints := make(map[string]Stint)

		for i, stint := range referenceTimingAppData.Lines[key].Stints {
			stints[strconv.Itoa(i)] = stint
		}

		timingAppData[key] = DriverTimingAppData{
			GridPos:      data.GridPos,
			Line:         data.Line,
			RacingNumber: data.RacingNumber,
			Stints:       stints,
		}
	}

	tad := TimingAppDataEvent{
		Data: timingAppData,
	}

	c.TimingAppDataChannel <- tad
}

func (c *Client) writeChangetoTimingAppDataChannel(m any) {
	if c.TimingAppDataChannel == nil {
		// The consumer did not ask for timing data events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var changeTimingAppData ChangeTimingAppData
	err = json.Unmarshal(s, &changeTimingAppData)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	tad := TimingAppDataEvent{
		Data: changeTimingAppData.Lines,
	}

	c.TimingAppDataChannel <- tad
}
