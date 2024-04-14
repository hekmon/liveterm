package termlive

import (
	"io"
	"os"
	"sync"
	"time"
)

var (
	// Config (must be changed before calling Start())
	RefreshInterval = 100 * time.Millisecond // RefreshInterval is the default refresh interval to update the ui
	UseStdErr       = false                  // use StdErr instead of StdOut
)

var (
	out         io.Writer
	ticker      *time.Ticker
	tdone       chan bool
	getterLines func() []string
	getterLine  func() string
	getterRaw   func() []byte
	mtx         sync.Mutex
)

// ForceUpdate forces an update of the terminal even if out of tick
func ForceUpdate() {
	mtx.Lock()
	update()
	mtx.Unlock()
}

// SetMultiLinesDataFx sets the function that returns the data to be displayed in the terminal.
// There is no need to end each line with a '\n' as it will be added automatically.
func SetMultiLinesUpdateFx(fx func() []string) {
	mtx.Lock()
	getterLines = fx
	getterLine = nil
	getterRaw = nil
	mtx.Unlock()
}

// SetMultiLinesDataFx sets the function that returns the data to be displayed in the terminal.
// There is no need to end each line with a '\n' as it will be added automatically.
func SetSingleLineUpdateFx(fx func() string) {
	mtx.Lock()
	getterLines = nil
	getterLine = fx
	getterRaw = nil
	mtx.Unlock()
}

// SetMultiLinesDataFx sets the function that returns the data to be displayed in the terminal.
func SetRawUpdateFx(fx func() []byte) {
	mtx.Lock()
	getterLines = nil
	getterLine = nil
	getterRaw = fx
	mtx.Unlock()
}

// Start starts the updater in a non-blocking manner.
// After calling Start(), the output (stdout or stderr) should not be used directly anymore.
func Start() {
	defer mtx.Unlock()
	mtx.Lock()
	// Nullify multiples calls to start
	if ticker != nil {
		return
	}
	// Start the updater
	if UseStdErr {
		out = os.Stderr
	} else {
		out = os.Stdout
	}
	ticker = time.NewTicker(RefreshInterval)
	tdone = make(chan bool)
	startListeningForTermResize()
	go worker()
}

// Stop stops the worker that updates the terminal.
// Clear will erase dynamic data from the terminal before stopping, otherwise it will update term one last time before stopping.
// Choosen output (stdout or stderr) can be used again directly after this call.
func Stop(clear bool) {
	tdone <- clear
	<-tdone
}

func worker() {
	var clear bool
	for {
		select {
		case <-ticker.C:
			mtx.Lock()
			update()
			mtx.Unlock()
		case <-termSizeChan:
			mtx.Lock()
			termSize = getTermSize()
			mtx.Unlock()
		case clear = <-tdone:
			mtx.Lock()
			ticker.Stop()
			ticker = nil
			stopListeningForTermResize()
			if clear {
				erase()
			} else {
				// update ui one last time with latest possible data
				update()
			}
			out = nil
			getterLines = nil
			getterLine = nil
			getterRaw = nil
			buf.Reset()
			lineCount = 0
			close(tdone)
			mtx.Unlock()
			return
		}
	}
}
