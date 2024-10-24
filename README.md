## F1 LiveTiming API Messages

### `CarData.z`

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

### `DriverList`

First message contains intrinsic data about the drivers and teams. Subsequent messages contain only grid position and only includes deltas (i.e. it will only include drivers that have changed position from the last message).

#### Example

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

### `ExtrapolatedClock`

Estimates how much time is left in the race based on laps remaining and pace of the leader (or max time of 2 hours).

#### Example

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

### `Heartbeat`

A recurring message letting the client know that the API is functioning

#### Example

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

### `LapCount`

Message sent when lap count increases; follows the lap of the lead car.

```json
[
  "LapCount",
  {
    "CurrentLap": 6
  },
  "2024-10-20T19:14:32Z"
]
```

### `Position.z`

Deflate-compressed statistics about car positions on track:

- Status (e.g. `OnTrack`)
- X
- Y
- Z

#### Example

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

### `RaceControlMessages`

Messages from race control, e.g. changing track conditions, flag information, incident investigation, etc. Messages will contain some or all of the fields:

- Lap
- Category
- Flag
- Scope
- Sector
- Status
- Message

#### Example

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

> ???

### `SessionData`

> TODO

### `SessionInfo`

> TODO

### `TimingAppData`

> TODO

### `TimingData`

> TODO

### `TimingStats`

> TODO

### `TopThree`

> TODO

### `TrackStatus`

Sent when the overall track status changes and includes the following statuses:

- AllClear
- Red
- SCDeployed
- Yellow

#### Example

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

### `WeatherData`

Current weather conditions at the track that includes the following data:

- AirTemp
- Humidity
- Pressure
- Rainfall
- TrackTemp
- WindDirection
- WindSpeed

#### Example

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