package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bcdxn/go-f1/f1livetiming"
)

func main() {
	i := make(chan os.Signal, 1)
	signal.Notify(i, os.Interrupt)
	d := make(chan struct{})
	weatherEvents := make(chan f1livetiming.WeatherDataEvent)

	c := f1livetiming.NewClient(i, d, f1livetiming.WithWeatherEvents(weatherEvents))
	err := c.Negotiate()
	if err != nil {
		panic(err)
	}
	// fmt.Println(c.ConnectionToken)
	go c.Connect()
	<-d // wait
	fmt.Println("done!")
}
