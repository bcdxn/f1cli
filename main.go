package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bcdxn/go-f1/f1livetiming"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	i := make(chan struct{})
	d := make(chan error)
	weatherEvents := make(chan f1livetiming.WeatherDataEvent)

	c := f1livetiming.NewClient(i, d, f1livetiming.WithWeatherChannel(weatherEvents))
	err := c.Negotiate()
	if err != nil {
		panic(err)
	}

	go c.Connect()
	<-interrupt // wait for interrupt OS signal
	close(i)    // notify client of interrupt
	<-d         // wait for client to gracefully close connection
	fmt.Println("done!")
}
