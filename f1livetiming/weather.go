package f1livetiming

type WeatherData struct {
	AirTemp       string `json:"AirTemp"`
	Humidity      string `json:"Humidity"`
	Pressure      string `json:"Pressure"`
	Rainfall      string `json:"Rainfall"`
	TrackTemp     string `json:"TrackTemp"`
	WindDirection string `json:"WindDirection"`
	WindSpeed     string `json:"WindSpeed"`
}

type WeatherDataEvent struct {
	Data WeatherData
}

func (c *Client) writeToWeatherChannel(m F1NestedMessage) {
	if c.WeatherChannel == nil {
		// The consumer did not ask for weather events; no need to process
		return
	}

	wde := WeatherDataEvent{
		Data: WeatherData{},
	}

	messageMap, ok := m.Arguments[1].(map[string]any)
	if !ok {
		// The message is in an unknown format; stop processing
		return
	}

	if v, ok := messageMap["AirTemp"].(string); ok {
		wde.Data.AirTemp = v
	}
	if v, ok := messageMap["Humidity"].(string); ok {
		wde.Data.Humidity = v
	}
	if v, ok := messageMap["Pressure"].(string); ok {
		wde.Data.Pressure = v
	}
	if v, ok := messageMap["Rainfall"].(string); ok {
		wde.Data.Rainfall = v
	}
	if v, ok := messageMap["TrackTemp"].(string); ok {
		wde.Data.TrackTemp = v
	}
	if v, ok := messageMap["WindDirection"].(string); ok {
		wde.Data.WindDirection = v
	}
	if v, ok := messageMap["WindSpeed"].(string); ok {
		wde.Data.WindSpeed = v
	}

	c.WeatherChannel <- wde
}
