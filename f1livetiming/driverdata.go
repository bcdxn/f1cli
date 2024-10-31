package f1livetiming

import "encoding/json"

type DriverListEvent struct {
	Data map[string]DriverData
}

func (c *Client) writeToDriverListChannel(m any) {
	if c.DriverListChannel == nil {
		// The consumer did not ask for driver data events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var driverData map[string]DriverData
	err = json.Unmarshal(s, &driverData)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	dde := DriverListEvent{
		Data: driverData,
	}

	c.DriverListChannel <- dde
}
