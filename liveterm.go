package liveterm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
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
	// state
	termOutput   *termenv.Output
	terminalFile termenv.File
	termRestore  func() error
	ticker       *time.Ticker
	waitUntil    time.Time
	buf, lastBuf bytes.Buffer
	getterLines  func() []string
	getterLine   func() string
	getterRaw    func() []byte
	mtx          sync.Mutex
	tdone        chan bool
	// missing cursor movement from termenv
	moveCursorBeginningOfTheLine = fmt.Sprintf(termenv.CSI+termenv.CursorHorizontalSeq, 0)
)

// ForceUpdate forces an update of the terminal with dynamic data between ticks.
func ForceUpdate() {
	mtx.Lock()
	update()
	mtx.Unlock()
}

// GetTermProfile returns the termenv profile used by liveterm.
// It can be used to create styles and colors that will be compatible with the terminal within your updater function.
// Only call this function after Start() has been called.
func GetTermProfil() termenv.Profile {
	return termOutput.Profile
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
	// Init term
	termOutput = termenv.NewOutput(Output)
	terminalFile = termOutput.TTY()
	if terminalFile == nil {
		termOutput = nil
		return errors.New("output is not a terminal")
	}
	if termRestore, err = termenv.EnableVirtualTerminalProcessing(termOutput); err != nil {
		return fmt.Errorf("failed to enable virtual terminal processing: %w", err)
	}
	initTermSize()
	// Start the updater
	ticker = time.NewTicker(RefreshInterval)
	tdone = make(chan bool)
	go worker()
	return
}

// Stop stops the worker that updates the terminal.
// Clear will erase dynamic data from the terminal before stopping, otherwise it will update term one last time before stopping.
// Choosen output (stdout or stderr) can be used again directly after this call.
func Stop(clear bool) (err error) {
	tdone <- clear
	<-tdone
	err = termRestore()
	termRestore = nil
	return
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
			termOutput = nil
			terminalFile = nil
			getterLines = nil
			getterLine = nil
			getterRaw = nil
			waitCtx = nil
			waitCtxCancel = nil
			delayedStopSignal = nil
			clearTermSize()
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
	if (getterLines == nil && getterLine == nil && getterRaw == nil) || termOutput == nil {
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
	_, _ = termOutput.Write(buf.Bytes())
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
	_, _ = termOutput.WriteString(moveCursorBeginningOfTheLine)
	termOutput.ClearLine()
	termOutput.ClearLines(linesCount)
}
