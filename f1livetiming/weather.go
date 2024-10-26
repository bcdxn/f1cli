package f1livetiming

import "encoding/json"

type WeatherDataEvent struct {
	Data WeatherData
}

func (c *Client) writeToWeatherChannel(m any) {
	if c.WeatherChannel == nil {
		// The consumer did not ask for weather events; no need to process
		return
	}

	s, err := json.Marshal(m)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	var wd WeatherData
	err = json.Unmarshal(s, &wd)
	if err != nil {
		// The message is in an unknown format; stop processing
		return
	}

	wde := WeatherDataEvent{
		Data: wd,
	}

	c.WeatherChannel <- wde
}
