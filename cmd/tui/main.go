package main

import (
	"context"
	"sync"

	"github.com/bcdxn/f1cli/internal/f1livetiming"
	"github.com/bcdxn/f1cli/internal/logger"
	"github.com/bcdxn/f1cli/internal/tui"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	l, f := logger.New()
	defer f.Close()
	// Create a wait group that ensures both client *and* TUI exit gracefully if either exits
	wg := sync.WaitGroup{}
	// create client responsible for listening to messags from the F1 LiveTiming API
	client := f1livetiming.New(f1livetiming.WithLogger(l))
	wg.Add(1)
	go func() {
		defer cancelCtx() // cancel the shared context between TUI and Client if either exits
		defer wg.Done()   // decrement shared wit group between TUI and Client
		client.Listen(ctx)
		l.Debug("client exited")
	}()
	// create TUI
	leaderboard := tui.NewLeaderboard(tui.WithContext(ctx), tui.WithLogger(l))
	wg.Add(1)
	go func() {
		defer cancelCtx() // cancel the shared context between TUI and Client if either exits
		defer wg.Done()   // decrement shared wit group between TUI and Client
		leaderboard.Run()
		l.Debug("tui exited")
	}()
	var err error
	// pass messages between client and TUI
	for {
		select {
		case <-ctx.Done():
			l.Debug("context done")
			wg.Wait()
			return
		case err = <-client.Done():
			if err != nil {
				l.Error("Client exited with error", "err", err)
			}
		case drivers := <-client.Drivers():
			leaderboard.Send(tui.DriversMsg(drivers))
		case meeting := <-client.Meeting():
			leaderboard.Send(tui.MeetingMsg(meeting))
		case raceCtrlMessages := <-client.RaceCtrlMsgs():
			leaderboard.Send(tui.RaceCtrlMsgsMsg(raceCtrlMessages))
		}
	}
}
