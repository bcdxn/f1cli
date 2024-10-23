package f1livetiming

type RaceControlData struct {
	Lap      uint8  `json:"Lap"`
	Category string `json:"Category"`
	Flag     string `json:"Flag"`
	Scope    string `json:"Scope"`
	Sector   uint8  `json:"Sector"`
	Status   string `json:"Status"`
	Message  string `json:"Message"`
}

type RaceControlEvent struct {
	Data RaceControlData
}

func (c *Client) writeToRaceControlChannel(m F1NestedMessage) {
	if c.RaceControlChannel == nil {
		// The consumer did not ask for race control events; no need to process
		return
	}
	messageMap, ok := m.Arguments[1].(map[string]interface{})
	if !ok {
		// The message is in an unknown format; stop processing
		return
	}

	msgs, ok := messageMap["Messages"].(map[string]interface{})
	if !ok {
		// The message is in an unknown format; stop processing
		return
	}

	for _, msg := range msgs {
		rce := RaceControlEvent{
			Data: RaceControlData{},
		}
		if strMap, ok := msg.(map[string]any); ok {
			if v, ok := strMap["Lap"].(float64); ok {
				rce.Data.Lap = uint8(v)
			}
			if v, ok := strMap["Category"].(string); ok {
				rce.Data.Category = v
			}
			if v, ok := strMap["Flag"].(string); ok {
				rce.Data.Flag = v
			}
			if v, ok := strMap["Scope"].(string); ok {
				rce.Data.Scope = v
			}
			if v, ok := strMap["Sector"].(float64); ok {
				rce.Data.Sector = uint8(v)
			}
			if v, ok := strMap["Status"].(string); ok {
				rce.Data.Status = v
			}
			if v, ok := strMap["Message"].(string); ok {
				rce.Data.Message = v
			}
		}

		c.RaceControlChannel <- rce
	}
}
