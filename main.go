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
	sessionData := make(chan f1livetiming.SessionDataEvent)

	p := tea.NewProgram(tui.NewModel(tuiLogger, i, d), tea.WithAltScreen())
	f1 := f1livetiming.NewClient(
		i,
		d,
		f1livetiming.WithSessionInfoChannel(sessionInfo),
		f1livetiming.WithDriverListChannel(driverList),
		f1livetiming.WithLapCountChannel(lapCount),
		f1livetiming.WithTimingDataChannel(timingData),
		f1livetiming.WithSessionDataChannel(sessionData),
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
				tuiLogger.Debug("sending tui.SessionInfoMsg")
				p.Send(tui.SessionInfoMsg{
					SessionInfo: si.Data,
				})
			case dl := <-driverList:
				f1Logger.Debug("received driverList channel update")
				tuiLogger.Debug("sending tui.DriverListMsg")
				p.Send(tui.DriverListMsg{
					DriverList: dl.Data,
				})
			case lc := <-lapCount:
				f1Logger.Debug("recevied lapCount channel update")
				tuiLogger.Debug("sending tui.LapCountMsg")
				p.Send(tui.LapCountMsg{
					LapCount: lc.Data,
				})
			case td := <-timingData:
				f1Logger.Debug("recevied timingData channel update")
				tuiLogger.Debug("sending tui.TimingDataMsg")
				p.Send(tui.TimingDataMsg{
					TimingData: td.Data.Lines,
				})
			case sd := <-sessionData:
				f1Logger.Debug("received sessionData channel update")
				tuiLogger.Debug("sending tui.SessionDataMsg")
				p.Send(tui.SessionDataMsg{
					SessionData: sd.Data,
				})
			}
		}

	}()

	_, err = p.Run()
	if err != nil {
		log.Fatal("Error starting TUI:", err.Error())
	}
}
