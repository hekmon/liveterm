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
	Out             = os.Stdout              // Out is the default output writer used by Start()
	// Internal
	termWidth       int
	overFlowHandled bool
	out             io.Writer
	getData         func() []byte
	ticker          *time.Ticker
	tdone           chan bool
	state           []byte
	mtx             sync.Mutex
	lineCount       int
)

func init() {
	termWidth, _ = getTermSize()
	if termWidth != 0 {
		overFlowHandled = true
	}
	// In case ForceUpdate() is called without Start()
	out = Out
}

// SetUpdater sets the function that returns the data to be displayed in the terminal
func SetUpdateFx(fx func() []byte) {
	mtx.Lock()
	getData = fx
	mtx.Unlock()
}

// Start starts the updater in a non-blocking manner
func Start() {
	mtx.Lock()
	// Nullify multiples calls to start
	if ticker != nil {
		mtx.Unlock()
		return
	}
	// Start the updater
	out = Out
	ticker = time.NewTicker(RefreshInterval)
	tdone = make(chan bool)
	go worker()
	mtx.Unlock()
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
			if ticker == nil {
				mtx.Unlock()
				continue
			}
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

// update is unsafe ! It must be called within a mutex lock by its parent
func update() {
	if getData == nil {
		return
	}
	data := getData()
	// take ownership of the data
	state = make([]byte, len(data))
	copy(state, data)
	// update terms
	erase()
	_, _ = write()
}

// erase is unsafe ! It must be called within a mutex lock by its parent
func erase() {
	clearLines()
	lineCount = 0
}

// write is unsafe ! It must be called within a mutex lock by its parent
func write() (n int, err error) {
	// do nothing if buffer is empty
	if len(state) == 0 {
		return
	}
	// Count the number of lines we are about to write for futur clearLines() calls
	var currentLine bytes.Buffer
	for _, b := range state {
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
	return out.Write(state)
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
