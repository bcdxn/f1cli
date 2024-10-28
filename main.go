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

	p := tea.NewProgram(tui.NewModel(tuiLogger, i, d), tea.WithAltScreen())
	f1 := f1livetiming.NewClient(
		i,
		d,
		f1livetiming.WithSessionInfoChannel(sessionInfo),
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
				f1Logger.Debug("sending sessionInfo tea message")
				p.Send(tui.SessionInfoMsg{
					SessionInfo: si.Data,
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
