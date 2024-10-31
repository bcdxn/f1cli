package main

import (
	"log"

	"github.com/bcdxn/go-f1/f1livetiming"
	"github.com/bcdxn/go-f1/tealogger"
	"github.com/bcdxn/go-f1/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tuiLogger := tealogger.New("TUI", tealogger.WithDebugOn())
	f1Logger := tealogger.New("F1C", tealogger.WithDebugOn())
	i := make(chan struct{})
	d := make(chan error)
	sessionInfo := make(chan f1livetiming.SessionInfoEvent)
	driverList := make(chan f1livetiming.DriverListEvent)
	lapCount := make(chan f1livetiming.LapCountEvent)
	timingData := make(chan f1livetiming.TimingDataEvent)

	p := tea.NewProgram(tui.NewModel(tuiLogger, i, d), tea.WithAltScreen())
	f1 := f1livetiming.NewClient(
		i,
		d,
		f1livetiming.WithSessionInfoChannel(sessionInfo),
		f1livetiming.WithDriverListChannel(driverList),
		f1livetiming.WithLapCountChannel(lapCount),
		f1livetiming.WithTimingDataChannel(timingData),
		f1livetiming.WithLogger(f1Logger),
	)

	err := f1.Negotiate()
	if err != nil {
		f1Logger.Error(err.Error())
		p.Send(tui.ErrorMsg{
			Err: err,
		})
		f1Logger.Debug("sending interrupt to f1 client")
		// Send interrupt to f1 livetiming client
		close(i)
		f1Logger.Debug("sending done message to TUI")
		p.Send(tui.DoneMsg{})
		return
	}

	go f1.Connect()

	go func() {
		listening := true
		for listening {
			select {
			case err = <-d:
				listening = false
				if err != nil {
					f1Logger.Error("error: ", err)
					p.Send(tui.ErrorMsg{
						Err: err,
					})
				}
				p.Send(tui.DoneMsg{})
			case si := <-sessionInfo:
				f1Logger.Debug("received sessionInfo channel update")
				p.Send(tui.SessionInfoMsg{
					SessionInfo: si.Data,
				})
			case dl := <-driverList:
				f1Logger.Debug("received driverList channel update")
				p.Send(tui.DriverListMsg{
					DriverList: dl.Data,
				})
			case lc := <-lapCount:
				f1Logger.Debug(("recevied lapCount channel update"))
				p.Send(tui.LapCountMsg{
					LapCount: lc.Data,
				})
			case td := <-timingData:
				f1Logger.Debug(("recevied timingData channel update"))
				p.Send(tui.TimingDataMsg{
					TimingData: td.Data.Lines,
				})
			}
		}

	}()

	_, err = p.Run()
	if err != nil {
		log.Fatal("Error starting TUI:", err.Error())
	}

	// i := make(chan struct{})
	// d := make(chan error)
	// weatherEvents := make(chan f1livetiming.WeatherDataEvent)

	// c := f1livetiming.NewClient(i, d, f1livetiming.WithWeatherChannel(weatherEvents))
	// err := c.Negotiate()
	// if err != nil {
	// 	panic(err)
	// }

	// go c.Connect()
	// <-interrupt // wait for interrupt OS signal
	// close(i)    // notify client of interrupt
	// <-d         // wait for client to gracefully close connection
	// fmt.Println("done!")
}
