package liveterm

import "os"

var (
	termCols           int
	termRows           int
	overFlowHandled    bool
	termSizeAutoUpdate bool
	termSizeChan       chan os.Signal
)

func init() {
	// Determine if overflow must be handled
	termCols, termRows = getTermSize()
	if termCols != 0 {
		overFlowHandled = true
	}
}

// TermSize represents the size of a terminal by its number of columns and rows.
type TermSize struct {
	Cols int
	Rows int
}

// GetTermSize returns the last known terminal size.
// It is either updated automatically on terminal resize on Unix like systems
// or updated at each refresh/update interval for windows.
func GetTermSize() (cols, rows int) {
	return termCols, termRows
}

// ForceTermSizeUpdate forces an update of the terminal size. This should not be necessary between Start() and Stop().
func ForceTermSizeUpdate() (cols, rows int) {
	mtx.Lock()
	termCols, termRows = getTermSize()
	cols = termCols
	rows = termRows
	mtx.Unlock()
	return
}
