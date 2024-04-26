package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hekmon/liveterm/v2"
	"github.com/muesli/termenv"
)

type postalCounter struct {
	counter int
}

func (pc *postalCounter) GetCounter() string {
	return strconv.Itoa(pc.counter)
}

func (pc *postalCounter) GetCenteredRedCounter() (output []string) {
	// Let's center the counter within the terminal window
	// It will adjust even if the terminal is resized
	cols, rows := liveterm.GetTermSize()
	output = make([]string, rows-1)
	counterStr := strconv.Itoa(pc.counter)
	output[len(output)/2] = fmt.Sprintf("%*s%s", (cols-len(counterStr))/2, "",
		liveterm.GetTermProfile().String(counterStr).Foreground(termenv.ANSIRed).Bold().String())
	return
}

func (pc *postalCounter) GetRawCounter() []byte {
	return []byte(strconv.Itoa(pc.counter))
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
	// This is a simple example to demonstrate the use of liveterm
	// The example will display a counter that increments every 10 milliseconds while updating terminal every 100 milliseconds

	// Change default configuration if needed
	liveterm.RefreshInterval = 100 * time.Millisecond
	liveterm.Output = os.Stdout

	// Start our wild counter
	pcDone := make(chan struct{})
	pc := postalCounter{}
	go func() {
		pc.StartCounting(3 * time.Second)
		close(pcDone)
	}()

	// Set the function that will return the data to be displayed
	// This can be done or changed even after Start() has been called
	// Try them out by uncommenting the one you want to use (and only one at a time)
	// liveterm.SetSingleLineUpdateFx(pc.GetCounter)
	liveterm.SetMultiLinesUpdateFx(pc.GetCenteredRedCounter)
	// liveterm.SetRawUpdateFx(pc.GetRawCounter)

	// Start live printing
	if err := liveterm.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start liveterm: %s\n", err)
		os.Exit(1)
	}

	// Let's write something to stdout while liveterm is running
	time.Sleep(2 * time.Second)
	fmt.Fprintf(liveterm.Bypass(), "This is a message that will be displayed on stdout while the counter is running\n")

	// Wait for the counter to finish
	<-pcDone

	// Release stdout
	if err := liveterm.Stop(false); err != nil {
		panic(err)
	}
}
