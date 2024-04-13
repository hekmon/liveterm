package termlive

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

// ESC is the ASCII code for escape character
const ESC = 27

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
	getter          func() []string
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

// SetMultiLinesDataFx sets the function that returns the data to be displayed in the terminal
func SetMultiLinesDataFx(fx func() []string) {
	mtx.Lock()
	getter = fx
	mtx.Unlock()
}

// SetMultiLinesDataFx sets the function that returns the data to be displayed in the terminal
func SetSingleLineDataFx(fx func() string) {
	mtx.Lock()
	getter = func() []string {
		return []string{fx()}
	}
	mtx.Unlock()
}

// Start starts the updater in a non-blocking manner
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

// Stop stops the updater that updates the terminal
func Stop() {
	tdone <- true
	<-tdone
}

// ForceUpdate forces an update of the terminal even if out of tick
func ForceUpdate() {
	mtx.Lock()
	update()
	mtx.Unlock()
}

func worker() {
	for {
		select {
		case <-ticker.C:
			mtx.Lock()
			update()
			mtx.Unlock()
		case <-tdone:
			mtx.Lock()
			ticker.Stop()
			ticker = nil
			update() // update the data one last time
			close(tdone)
			mtx.Unlock()
			return
		}
	}
}

// update is unsafe ! It must be called within a mutex lock by one of its callers
func update() {
	if getter == nil || out == nil {
		return
	}
	buf.Reset()
	for _, line := range getter() {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
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
	// do nothing if buffer is empty
	if buf.Len() == 0 {
		return
	}
	// Count the number of actual term lines we are about to write for futur clearLines() calls
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
