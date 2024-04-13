package termlive

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

// ESC is the ASCII code for escape character
const esc = 27

var (
	// Config
	RefreshInterval = 100 * time.Millisecond // RefreshInterval is the default refresh interval to update the ui
	UseStdErr       = false                  // use StdErr instead of StdOut
	// Internal
	termWidth       int
	overFlowHandled bool
	out             io.Writer
	ticker          *time.Ticker
	tdone           chan bool
	getterLines     func() []string
	getterLine      func() string
	getterRaw       func() []byte
	buf             bytes.Buffer
	mtx             sync.Mutex
	lineCount       int
)

func init() {
	termWidth, _ = getTermSize()
	if termWidth != 0 {
		overFlowHandled = true
	}
}

// GetTermWidth returns the current terminal width
func GetTermWidth() int {
	return termWidth
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
// You are responsible for adding the trailing '\n' at the end.
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
	go worker()
}

// Stop stops the updater that updates the terminal. Clear will erase dynamic data from the terminal before stopping.
// Choosen output (stdout or stderr) can be used again directly after this call.
func Stop(clear bool) {
	tdone <- clear
	<-tdone
}

// ForceUpdate forces an update of the terminal even if out of tick
func ForceUpdate() {
	mtx.Lock()
	update()
	mtx.Unlock()
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
			if clear {
				erase()
			} else {
				update() // update ui one last time with latest possible data
			}
			ticker = nil
			out = nil
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
	// Rebuild buffer with current data
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
	// Update terminal with it
	erase()
	_, _ = write()
}

// erase is unsafe ! It must be called within a mutex lock by one of its callers
func erase() {
	clearLines()
	lineCount = 0
}

// write is unsafe ! It must be called within a mutex lock by one of its callers
func write() (n int, err error) {
	// Count the number of actual term lines we are about to write for futur clearLines() call
	var currentLine bytes.Buffer
	for _, b := range buf.Bytes() {
		if b == '\n' {
			lineCount++
			currentLine.Reset()
		} else if overFlowHandled {
			currentLine.Write([]byte{b})
			if currentLine.Len() > termWidth {
				lineCount++
				currentLine.Reset()
			}
		}
	}
	// Write the current state
	return out.Write(buf.Bytes())
}

// Bypass creates an io.Writer which allows to write a permalent lines to the terminal. Do not forget to include a final '\n' when writting to it.
func Bypass() io.Writer {
	return &bypass{}
}

type bypass struct{}

// Each write will retrigger the update of the previous dynamic data even if out of tick.
func (bypass) Write(p []byte) (n int, err error) {
	defer mtx.Unlock()
	mtx.Lock()
	// erase current dynamic data
	erase()
	// write permanent data
	if n, err = out.Write(p); err != nil {
		return
	}
	// rewrite dynamic data after it
	_, err = write()
	return
}
