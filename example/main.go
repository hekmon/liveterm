package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hekmon/termlive"
)

type postalCounter struct {
	counter int
}

func (pc *postalCounter) GetCounter() string {
	return strconv.Itoa(pc.counter)
}

func (pc *postalCounter) StartCounting(duration time.Duration) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(duration)
	for {
		select {
		case <-ticker.C:
			pc.counter++
		case <-timer.C:
			return
		}
	}
}

func main() {
	// This is a simple example to demonstrate the use of termlive
	// The example will display a counter that increments every 10 milliseconds while updating terminal every 100 milliseconds

	// Change default configuration if needed
	termlive.RefreshInterval = 100 * time.Millisecond
	termlive.UseStdErr = false

	// Start our wild counter
	pcDone := make(chan struct{})
	pc := postalCounter{}
	go func() {
		pc.StartCounting(3 * time.Second)
		close(pcDone)
	}()

	// Set the function that will return the data to be displayed
	// This can be done or changed even after Start() has been called
	termlive.SetSingleLineUpdateFx(pc.GetCounter)

	// Start live printing
	termlive.Start()

	// Let's write something to stdout while termlive is running
	time.Sleep(2 * time.Second)
	fmt.Fprintf(termlive.Bypass(), "This is a message that will be displayed on stdout while the counter is running\n")

	// Wait for the counter to finish
	<-pcDone

	// Release stdout
	termlive.Stop(false)
}
