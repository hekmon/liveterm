package liveterm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
)

const (
	resizeWait = 500 * time.Millisecond
)

var (
	// Config (must be changed before calling Start())
	RefreshInterval           = 100 * time.Millisecond // RefreshInterval defines the time between each output refresh
	Output          io.Writer = os.Stdout              // Terminal Output
)

var (
	out          io.Writer
	ticker       *time.Ticker
	waitUntil    time.Time
	buf, lastBuf bytes.Buffer
	getterLines  func() []string
	getterLine   func() string
	getterRaw    func() []byte
	mtx          sync.Mutex
	tdone        chan bool
)

// ForceUpdate forces an update of the terminal with dynamic data between ticks.
func ForceUpdate() {
	mtx.Lock()
	update()
	mtx.Unlock()
}

// GetTermSize returns the last known terminal size.
// It is either updated automatically on terminal resize on Unix like systems
// or updated at each refresh/update interval for windows.
func GetTermSize() (cols, rows int) {
	return termCols, termRows
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
func Start() (err error) {
	defer mtx.Unlock()
	mtx.Lock()
	// Nullify multiples calls to start
	if ticker != nil {
		return errors.New("liveterm is already started")
	}
	// Try to open the terminal to gets its informations
	if err = initTermInfos(); err != nil {
		return fmt.Errorf("failed to init terminal: %w", err)
	}
	// Start the updater
	out = Output
	ticker = time.NewTicker(RefreshInterval)
	tdone = make(chan bool)
	go worker()
	return
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
			// Avoid future ticks
			ticker.Stop()
			ticker = nil
			// Handle a potential delayed bypass writer
			if waitCtxCancel != nil {
				waitCtxCancel()
				<-delayedStopSignal
			}
			// Either clear or update one last time
			if clear {
				erase()
			} else {
				update()
			}
			// Cleanup
			out = nil
			getterLines = nil
			getterLine = nil
			getterRaw = nil
			waitCtx = nil
			waitCtxCancel = nil
			delayedStopSignal = nil
			_ = clearTermInfos()
			buf.Reset()
			lastBuf.Reset()
			close(tdone)
			mtx.Unlock()
			return
		}
	}
}

// update is unsafe ! It must be called within a mutex lock by one of its callers
func update() {
	if (getterLines == nil && getterLine == nil && getterRaw == nil) || out == nil {
		return
	}
	// Update terminal size for erase
	if overFlowHandled {
		termColsPrevious, termRowsPrevious = termCols, termRows
		termCols, termRows = getTermSize()
		if termCols != termColsPrevious || termRows != termRowsPrevious {
			// term has been resized, wait for stability before computing the lines to erase
			// in case the terminal resizing is not done yet
			waitUntil = time.Now().Add(resizeWait)
			return
		}
		// Size is stable between 2 ticks, but are we in a wait period ?
		// Ensure fast rize from the user is handled correctly and does not leave lines behind
		if waitUntil.After(time.Now()) {
			return
		}
	}
	// Build unused buffer with fresh data
	buf.Reset()
	switch {
	case getterLines != nil:
		for _, line := range getterLines() {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	case getterLine != nil:
		buf.WriteString(getterLine())
		buf.WriteByte('\n')
	case getterRaw != nil:
		buf.Write(getterRaw())
	}
	// Cleanup terminal based on previous data and current terminal size
	erase()
	// Update terminal with it
	_, _ = out.Write(buf.Bytes())
	// Swap buffers to minimze memory allocation
	lastBuf, buf = buf, lastBuf
}

// erase is unsafe ! It must be called within a mutex lock by one of its callers
func erase() {
	// Given the previous data we printed and the current terminal size
	// we can compute the number of lines to erase.
	linesCount := 0
	var currentLineWidth, runeWidth int
	for _, r := range lastBuf.String() {
		if r == '\n' {
			linesCount++
			currentLineWidth = 0
			continue
		}
		if overFlowHandled {
			runeWidth = runewidth.RuneWidth(r)
			currentLineWidth += runeWidth
			if currentLineWidth > termCols {
				linesCount++
				currentLineWidth = runeWidth
			}
		}
	}
	clearLines(linesCount)
}

/*
   Bypass related
*/

var (
	waitBuf           bytes.Buffer
	waitCtx           context.Context
	waitCtxCancel     context.CancelFunc
	delayedStopSignal chan struct{}
)

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
	// check if we are within a wait period
	if overFlowHandled && waitUntil.After(time.Now()) {
		// write it to a temporary buffer
		n, err = waitBuf.Write(p)
		// Start the delayer bypass writer if not already started
		if waitCtx == nil {
			waitCtx, waitCtxCancel = context.WithCancel(context.Background())
			delayedStopSignal = make(chan struct{})
			go delayedBypassWritter()
		}
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

func delayedBypassWritter() {
	defer close(delayedStopSignal)
loop:
	for {
		waitTime := time.NewTimer(time.Until(waitUntil))
		select {
		case <-waitTime.C:
			mtx.Lock()
			// In case wait duration has been reset during our wait
			if waitUntil.After(time.Now()) {
				continue loop
			}
			// Wait time is over, let's by pass
			erase()
			_, _ = out.Write(waitBuf.Bytes())
			_, _ = out.Write(lastBuf.Bytes())
			waitBuf.Reset()
			// Before exiting, mark ourself as not started
			waitCtx = nil
			waitCtxCancel = nil
			// Our work is done
			mtx.Unlock()
			return
		case <-waitCtx.Done():
			// liveterm is being stopped, let's flush the buffer
			// do not try to lock the mutex as it is being locked by the main worker who canceled our context
			erase()
			_, _ = out.Write(waitBuf.Bytes())
			_, _ = out.Write(lastBuf.Bytes())
			waitBuf.Reset()
			// Mark ourself as not started before exiting
			waitCtx = nil
			waitCtxCancel = nil
			return
		}
	}
}
