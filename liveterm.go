package liveterm

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

var (
	// Config (must be changed before calling Start())
	RefreshInterval           = 100 * time.Millisecond // RefreshInterval defines the time between each output refresh
	Output          io.Writer = os.Stdout              // Terminal Output
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

// ForceUpdate forces an update of the terminal with dynamic data between ticks.
func ForceUpdate() {
	mtx.Lock()
	update()
	mtx.Unlock()
}

// SetMultiLinesDataFx sets the function that will be called to get data update.
// There is no need to end each line with a '\n' as it will be added automatically.
func SetMultiLinesUpdateFx(fx func() []string) {
	mtx.Lock()
	getterLines = fx
	getterLine = nil
	getterRaw = nil
	mtx.Unlock()
}

// SetSingleLineUpdateFx sets the function that will be called to get data update.
// There is no need to end each line with a '\n' as it will be added automatically.
func SetSingleLineUpdateFx(fx func() string) {
	mtx.Lock()
	getterLines = nil
	getterLine = fx
	getterRaw = nil
	mtx.Unlock()
}

// SetRawUpdateFx sets the function that will be called to get data update.
func SetRawUpdateFx(fx func() []byte) {
	mtx.Lock()
	getterLines = nil
	getterLine = nil
	getterRaw = fx
	mtx.Unlock()
}

// Start starts the updater in a non-blocking manner.
// After calling Start(), the output (stdout or stderr) should not be used directly anymore.
// See Bypass() if you need to print regular things while liveterm is running.
func Start() {
	defer mtx.Unlock()
	mtx.Lock()
	// Nullify multiples calls to start
	if ticker != nil {
		return
	}
	// Start the updater
	out = Output
	ticker = time.NewTicker(RefreshInterval)
	tdone = make(chan bool)
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
		case clear = <-tdone:
			mtx.Lock()
			ticker.Stop()
			ticker = nil
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

// Bypass creates an io.Writer which allow to write permalent stuff to the terminal while liveterm is running.
// Do not forget to include a final '\n' when writting to it.
func Bypass() io.Writer {
	return bypass{}
}

type bypass struct{}

func (bypass) Write(p []byte) (n int, err error) {
	defer mtx.Unlock()
	mtx.Lock()
	// if liveterm is not started, out is nil
	if out == nil {
		err = errors.New("liveterm is not started, can not write to terminal")
		return
	}
	// erase current dynamic data
	erase()
	// write permanent data
	if n, err = out.Write(p); err != nil {
		return
	}
	// rewrite the last known dynamic data after it
	_, err = out.Write(lastBuf.Bytes())
	return
}
