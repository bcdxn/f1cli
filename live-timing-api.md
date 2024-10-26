## F1 LiveTiming API Messages

The F1 LiveTiming API is composed of messages sent from a [SignalR](https://learn.microsoft.com/aspnet/core/signalr) server over websockets.

There are two core types of messages:

1. Reference Data Message that contains an initial state for all of the data related to the subscriptions specified by the client. A single Reference Data Message is emitted at the start of the connection. The message is in JSON format where the top-level key is `"R"` and the properties of the reference data are the names of the subscriptions.
    ```json
    {
        "R": {
            "Heartbeat": {},
            "CarData.z": "",
            "Position.z": "",
            "ExtrapolatedClock": {},
            "TopThree": {},
            "TimingStats": {},
            "TimingAppData": {},
            "WeatherData": {},
            "TrackStatus": {},
            "DriverList": {},
            "RaceControlMessages": {},
            "SessionInfo": {},
            "SessionData": {},
            "LapCount": {},
            "TimingData": {}
        },
        // Interval in seconds of change data messages
        // requested by client
        "I": "5"
    }
    ```
2. Change Data Message that contains change updates to the reference data for each subscription. A Change Data Message is sent whenever there is a change to the relevant data; tens of thousands may be sent over the course of a session. The change data is nested within an `"M"` property as shown below:
    ```json
    {
        // possibly indicates a Signalr Group receiving
        // the same change data
        "C": "...", 
        // The property we're interested in that contains
        // change data
        "M": [ 
            {
            "H": "Streaming",
            "M": "feed",
            "A": [
                // Subscription name
                "<name of data>",
                // Can be JSON, or a string containing
                // compressed JSON using Deflate Compression
                "<data>", 
                "<timestamp>"
            ]
            }
        ]
    }
    ```

## Data Available for Subscription

- [CarData](#car-dataz)
- [DriverList](#driver-list)
- [ExtrapolatedClock](extrapolated-clock)
- [Heartbeat](#hearbeat)
- [LapCount](#lap-count)
- [Position.z](#positionz)
- [RaceControlMessages](#race-control-messages)
- [SessionData](#session-data)
- [SessionInfo](#session-info)
- [TopThree](#top-three)
- [TimingAppData](#timing-app-data)
- [TimingData](#timing-data)
- [TimingStats](#timing-stats)
- [TrackStatus](#track-status)
- [WeatherData](#weather-data)

### [`CarData.z`](#cardataz)

[Deflate-compressed](https://en.wikipedia.org/wiki/DEFLATE) statistics about individual cars including:

- RPM
- Speed
- Gear
- Throttle
- Brake
- DRS

#### Example Message

##### Compressed

```json
[
    "CarData.z",
    "1ZZNS8NAEIb/y55bma/9yrX4D/SieChSUJAeam8l/911K9ia2TLEptYcNiHkZWbezDy7O3e73m5eV++ue9y5++2z6xwByRxhjvmOsPO583LDBCgsD27mFstN+Xrn8HNZvCzX69VbfQGug5mjunJdpa5+/1xuqe/ry4FOACxSBEXLHrJFOzZf1BLmHL1FG7RiEZNFmzRtSFS1QlWMDTFpTmGMQ6cQpKrr/VtPqtMJDYkTq4blQ+1h3ghHar0/YmxE/qGOippQhmq9blY7LIRgqJu1DmPPYhkKzbNWdx6XLJphQ6ler/eq29xy+zBwsOWsB47ajxIkS+A0YpShL9fsFOIkAWLGqRDHOZmmXkectzTReREnQJZpbyHOFFdFnHwZ9d8QJyA8HnGlP+gvEed/gziLZ1eFOM5kadDzI67sgxYMTIO4UjR4ThMiznQoUhtQwuVPcQVxPBZxpYMsE6MjjmE/rYKnMTMN4sjylxqICzIecSdO+RdBHFgOFw3EmfazK0MctDaUaRFX9kHLfjAacU/9Bw==",
    "2024-10-19T21:59:54.9200538Z"
]
```

##### Decompressed

```json
[
  "CarData.z",
  {
    "Entries": [
      {
        "Utc": "2024-10-19T21:59:54.3201434Z",
        "Cars": {
          "1": {            // driver number
            "Channels": {
              "0": 0,       // RPM
              "2": 0,       // Speed
              "3": 0,       // Gear
              "4": 0,       // Break
              "5": 0,       // Throttle
              "45": 8       // Has DRS
            }
          },
          "4": {
            "Channels": {
              "0": 4000,
              "2": 0,
              "3": 0,
              "4": 0,
              "5": 0,
              "45": 8
            }
          }
          // ...
        }
      }
    ]
  },
  "2024-10-19T21:59:54.9200538Z"
]
```

### [`DriverList`](#driverlist)

First message contains intrinsic data about the drivers and teams. Subsequent messages contain only grid position and only includes deltas (i.e. it will only include drivers that have changed position from the last message).

#### Example Message

##### First Message With Intrinsic Data

```json
[
  "DriverList",
  {
    "4": {
      "RacingNumber": "4",
      "BroadcastName": "L NORRIS",
      "FullName": "Lando NORRIS",
      "Tla": "NOR",
      "Line": 1, // grid position
      "TeamName": "McLaren",
      "TeamColour": "FF8000",
      "FirstName": "Lando",
      "LastName": "Norris",
      "Reference": "LANNOR01",
      "HeadshotUrl": "https://media.formula1.com/d_driver_fallback_image.png/content/dam/fom-website/drivers/L/LANNOR01_Lando_Norris/lannor01.png.transform/1col/image.png",
      "CountryCode": "GBR"
    }
    // contains all 20 drivers
  },
  "2024-10-20T18:48:10.916Z"
]
```

##### Subsequent 'Delta' Messages

```json
[
  "DriverList",
  {
    "31": {
      "Line": 18
    },
    "23": {
      "Line": 19
    },
    "77": {
      "Line": 17
    }
  },
  "2024-10-20T19:04:22.604Z"
]
```

### [`ExtrapolatedClock`](#extrapolatedclock)

Estimates how much time is left in the race based on laps remaining and pace of the leader (or max time of 2 hours).

#### Example Message

```json
[
  "ExtrapolatedClock",
  {
    "Utc": "2024-10-20T19:03:49.011Z",
    "Remaining": "01:59:59",
    "Extrapolating": true
  },
  "2024-10-20T19:03:49.011Z"
]
```

### [`Heartbeat`](#hearbeat)

A recurring message letting the client know that the API is functioning

#### Example Message

```json
[
  "Heartbeat",
  {
    "Utc": "2024-10-20T19:03:56.292137Z",
    "_kf": true
  },
  "2024-10-20T19:03:55.219Z"
]
```

### [`LapCount`](#lapcount)

Message sent when lap count increases; follows the lap of the lead car.

#### Example Message

```json
[
  "LapCount",
  {
    "CurrentLap": 6
  },
  "2024-10-20T19:14:32Z"
]
```

### [`Position.z`](#positionz)

Deflate-compressed statistics about car positions on track:

- Status (e.g. `OnTrack`)
- X
- Y
- Z

#### Example Message

##### Compressed

```json
[
    "Position.z",
    "7ZZNS8NAEIb/y5xTmY/9zN2zgj1oxUORHoI0lTaeSv67Sd3U3YMjCN72UlLIw8w8O/uSM9wfTt3QHXpon8+w7va707Ddv0MLjGxWhCvGNYXWUkv+RpwYcXYDDdz2w7HbnaA9A80/D8N2+Jj+wl2/Pm5f36ZXHqFdTUADT9NDZN/ABloSlLEBozBG7BdD6OMC8QQRalSQhTJSUFp/7F2iGClvkLQOBUOiaOazWk6hrE9U9FhAQROInCBLOcSaC0/JRXBFe8zaUHHRTlhoZ9GGEly0oy8oTaBzqVbE4qzYa1MtZxUo5JBoKi6zXPoTvlJ2prS1+HGZjObCfK8FYrHtmgvPqcNgTV7KWk17SC6iKyCnntUi0Mc0FIcZ8pp1Z1J7kWxeKajXyoXrtcru/Tg2v4eMnTIDHdWQqSFTQ6aGzP+ETHAiDuuXTA2ZGjI1ZP4SMi/jJw==",
    "2024-10-20T18:51:17.8633605Z"
]
```

##### Decompressed

```json
[
  "Position.z",
  {
    // Positions array contains multiple 'Entries' sets in a single message that are within the
    // same second
    "Position": [
      {
        "Timestamp": "2024-10-20T18:51:17.3634365Z",
        "Entries": {
          "1": {
            "Status": "OnTrack",
            "X": -634,
            "Y": -927,
            "Z": 1303
          }
          // ...
          // each Entries block contains all 20 cars
        }
      }
    ]
  }
]
```

### [`RaceControlMessages`](#racecontrolmessages)

Messages from race control, e.g. changing track conditions, flag information, incident investigation, etc. Messages will contain some or all of the fields:

- Lap
- Category
- Flag
- Scope
- Sector
- Status
- Message

#### Example Message

```json
[
  "RaceControlMessages",
  {
    "Messages": {
      "8": { // each message is numbered
        "Utc": "2024-10-20T19:04:13",
        "Lap": 1,
        "Category": "Flag",
        "Flag": "YELLOW",
        "Scope": "Sector",
        "Sector": 3,
        "Message": "YELLOW IN TRACK SECTOR 3"
      }
    }
  },
  "2024-10-20T19:04:13.111Z"
]
```

```json
[
  "RaceControlMessages",
  {
    "Messages": {
      "55": {
        "Utc": "2024-10-20T19:42:31",
        "Lap": 22,
        "Category": "Other",
        "Message": "TURN 12 INCIDENT INVOLVING CARS 23 (ALB) AND 20 (MAG) NOTED - FORCING ANOTHER DRIVER OFF THE TRACK"
      }
    }
  },
  "2024-10-20T19:42:31.536Z"
]
```

```json
[
  "RaceControlMessages",
  {
    "Messages": {
      "70": {
        "Utc": "2024-10-20T20:06:30",
        "Lap": 37,
        "Category": "Flag",
        "Flag": "BLUE",
        "Scope": "Driver",
        "RacingNumber": "24",
        "Message": "WAVED BLUE FLAG FOR CAR 24 (ZHO) TIMED AT 15:06:30"
      }
    }
  },
  "2024-10-20T20:06:30.639Z"
]
```

### `RcmSeries`

> TODO

### [`SessionData`](#sessiondata)

In Qualifying it gives updates on Session 1, 2, 3. In all sessions it includes `Started` and `Finished` status messages. It also includes TrackStatus; appears redundant with `TrackStatus` message.

#### Example Message

```json
[
  "SessionData",
  {
    "StatusSeries": {
      "7": {
        "Utc": "2024-10-19T22:48:00.25Z",
        "SessionStatus": "Started"
      }
    }
  },
  "2024-10-19T22:48:00.25Z"
]
```

```json
[
  "SessionData",
  {
    "Series": {
      "2": {
        "Utc": "2024-10-19T22:46:00.154Z",
        "QualifyingPart": 3
      }
    }
  },
  "2024-10-19T22:46:00.154Z"
]
```

```json
[
  "SessionData",
  {
    "StatusSeries": {
      "8": {
        "Utc": "2024-10-19T22:59:57.399Z",
        "TrackStatus": "Yellow"
      }
    }
  },
  "2024-10-19T22:59:57.399Z"
]
```

### [`SessionInfo`](#sessioninfo)

Includes intrinsic data about the session and track. This event is only emitted on subscription as part of the larger "R"-type event and at the end when the session is completed.

#### Example Message

```json
[
  "SessionInfo",
  {
    "Meeting": {
      "Key": 1247,
      "Name": "United States Grand Prix",
      "OfficialName": "FORMULA 1 PIRELLI UNITED STATES GRAND PRIX 2024",
      "Location": "Austin",
      "Number": 19,
      "Country": {
        "Key": 19,
        "Code": "USA",
        "Name": "United States"
      },
      "Circuit": {
        "Key": 9,
        "ShortName": "Austin"
      }
    },
    "ArchiveStatus": {
      "Status": "Complete"
    },
    "Key": 9617,
    "Type": "Race",
    "Name": "Race",
    "StartDate": "2024-10-20T14:00:00",
    "EndDate": "2024-10-20T16:00:00",
    "GmtOffset": "-05:00:00",
    "Path": "2024/2024-10-20_United_States_Grand_Prix/2024-10-20_Race/",
    "_kf": true
  },
  "2024-10-20T20:42:06.544Z"
]
```

### [`TimingAppData`](#timingappdata)

Generally this message consists of data specific to a stint. It contains data about tire compound at the start a stint, laps within a stint, and 'last lap' time.

#### Example Message

```json
[
  "TimingAppData",
  {
    "Lines": {
      "55": {
        "Line": 6
      },
      "16": {
        "Line": 7
      },
      "4": {
        "Line": 8
      },
      "81": {
        "Line": 9
      },
      "14": {
        "Line": 10
      },
      "22": {
        "Line": 11
      },
      "30": {
        "Line": 12
      },
      "18": {
        "Line": 13
      },
      "77": {
        "Line": 4
      },
      "24": {
        "Line": 5
      }
    }
  },
  "2024-10-19T22:02:32.581Z"
]
```

```json
[
  "TimingAppData",
  {
    "Lines": {
      "14": {
        "Stints": {
          "0": {
            "TotalLaps": 4
          }
        }
      },
      "77": {
        "Stints": {
          "0": {
            "TotalLaps": 1
          }
        }
      }
    }
  },
  "2024-10-19T22:02:37.253Z"
]
```

```json
[
  "TimingAppData",
  {
    "Lines": {
      "1": {
        "Stints": {
          "0": {
            "TotalLaps": 5
          }
        }
      },
      "63": {
        "Stints": {
          "0": {
            "TotalLaps": 7
          }
        }
      }
      //...
    }
  },
  "2024-10-19T18:07:10.916Z"
]
```

```json
[
  "TimingAppData",
  {
    "Lines": {
      "16": {
        "Stints": {
          "0": {
            "LapTime": "1:38.655",
            "LapNumber": 2,
            "LapFlags": 1
          }
        }
      }
      // ...
    }
  },
  "2024-10-19T18:07:15.921Z"
]
```

```
[
  "TimingAppData",
  {
    "Lines": {
      "11": {
        "Stints": {
          "1": {
            "Compound": "HARD",
            "LapFlags": 0,
            "New": "true",
            "StartLaps": 0,
            "TotalLaps": 0,
            "TyresNotChanged": "0"
          }
        }
      }
    }
  },
  "2024-10-20T19:50:24.87Z"
]
```

### [`TimingData`](#timingdata)

Contains interval and leader gaps in seconds, speed trap values, personal best sector times/indicators, contains sector/segment status data. Statuses include:

- 0
- 64
- 2048
- 2049
- 2051
- 2064
- 4160

#### Example Message

```json
[
  "TimingData",
  {
    "Lines": {
      "55": {
        "GapToLeader": "+1.218",
        "IntervalToPositionAhead": {
          "Value": "+0.396"
        }
      }
    }
  },
  "2024-10-20T19:04:42.119Z"
]
```

```json
[
  "TimingData",
  {
    "Lines": {
      "30": {
        "Sectors": {
          "2": {
            "Segments": {
              "5": {
                "Status": 2048
              }
            }
          }
        }
      }
    }
  },
  "2024-10-19T18:03:12.333Z"
]
```

```json
[
  "TimingData",
  {
    "Lines": {
      "63": {
        "NumberOfLaps": 1,
        "Sectors": {
          "2": {
            "PersonalFastest": true,
            "Value": "34.482"
          }
        },
        "Speeds": {
          "FL": { // FL, I1, I2, ST
            "PersonalFastest": true,
            "Value": "202"
          }
        }
      }
    }
  },
  "2024-10-20T19:05:40.612Z"
]
```

```json
[
  "TimingData",
  {
    "Lines": {
      "11": {
        "Status": 256
      }
    }
  },
  "2024-10-19T22:05:58.159Z"
]
```

```json
[
  "TimingData",
  {
    "Lines": {
      "77": {
        "InPit": true,
        "Status": 80,
        "NumberOfPitStops": 1
      }
    }
  },
  "2024-10-19T22:05:59.809Z"
]
```

### [`TimingStats`](#timingstats)

Messages include best sector times, best lap times, and best speed trap speeds for each driver.

#### Example Message

```json
[
  "TimingStats",
  {
    "Lines": {
      "14": {
        "BestSectors": {
          "1": {
            "Value": "40.074"
          }
        }
      }
    }
  },
  "2024-10-20T19:53:24.698Z"
]
```

```json
[
  "TimingStats",
  {
    "Lines": {
      "27": {
        "PersonalBestLapTime": {
          "Position": 10
        }
      },
      "30": {
        "PersonalBestLapTime": {
          "Lap": 28,
          "Position": 9,
          "Value": "1:40.235"
        }
      }
    }
  },
  "2024-10-20T19:53:29.193Z"
]
```

```json
[
  "TimingStats",
  {
    "Lines": {
      "30": {
        "BestSpeeds": {
          "I1": {
            "Position": 1,
            "Value": "223"
          }
        }
      }
    }
  },
  "2024-10-20T19:53:56.002Z"
]
```

### [`TopThree`](#topthree)

> TODO

### [`TrackStatus`](#trackstatus)

Sent when the overall track status changes and includes the following statuses:

- AllClear
- Yellow
- SCDeployed
- Red
- VSCDeployed
- VSCEnding

#### Example Message

```json
[
  "TrackStatus",
  {
    "Status": "1",
    "Message": "AllClear",
    "_kf": true
  },
  "2024-10-20T19:21:45.067Z"
]
```

### [`WeatherData`](#weatherdata)

Current weather conditions at the track that includes the following data:

- AirTemp
- Humidity
- Pressure
- Rainfall
- TrackTemp
- WindDirection
- WindSpeed

#### Example Message

```json
[
  "WeatherData",
  {
    "AirTemp": "27.0",
    "Humidity": "47.0",
    "Pressure": "1007.6",
    "Rainfall": "0",
    "TrackTemp": "46.1",
    "WindDirection": "193",
    "WindSpeed": "2.1",
    "_kf": true
  },
  "2024-10-19T18:04:18.774Z"
]
```