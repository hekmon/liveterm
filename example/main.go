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

func (pc *postalCounter) GetCenteredCounter(termSize termlive.TermSize) (output []string) {
	// Let's center the counter within the terminal window
	// It will adjust even if the terminal is resized
	counterStr := strconv.Itoa(pc.counter)
	counterLen := len(counterStr)
	output = make([]string, termSize.Rows-1)
	for lineIndex := 0; lineIndex < len(output); lineIndex++ {
		if lineIndex == termSize.Rows/2 {
			output[lineIndex] = fmt.Sprintf("%*s%s", (termSize.Cols-counterLen)/2, "", counterStr)
		}
	}
	return
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
	// termlive.SetMultiLinesUpdateFx(func() []string {
	// 	return pc.GetCenteredCounter(termlive.GetTermSize())
	// })

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
