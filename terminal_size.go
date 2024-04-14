package termlive

import "os"

var (
	termSize           TermSize
	overFlowHandled    bool
	termSizeAutoUpdate bool
	termSizeChan       chan os.Signal
)

func init() {
	// Determine if overflow must be handled
	termSize = getTermSize()
	if termSize.Cols != 0 {
		overFlowHandled = true
	}
}

// TermSize represents the size of a terminal by its number of columns and rows.
type TermSize struct {
	Cols int
	Rows int
}

// GetTermSize returns the last known terminal size.
// It is either updated automatically on terminal resize on POSIX or updated at each refresh/update interval for windows.
func GetTermSize() TermSize {
	return termSize
}
