package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bcdxn/go-f1/f1livetiming"
)

func main() {
	fmt.Println("running...")

	i := make(chan os.Signal, 1)
	d := make(chan bool)

	signal.Notify(i, os.Interrupt)

	weatherEvents := make(chan f1livetiming.WeatherDataEvent)

	c := f1livetiming.NewClient(i, d, weatherEvents)
	c.Negotiate()
	go c.Connect()
	<-d // wait
	fmt.Println("done!")
}
