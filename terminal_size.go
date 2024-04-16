package liveterm

var (
	termCols, termColsPrevious int
	termRows, termRowsPrevious int
	overFlowHandled            bool
)

func init() {
	// Determine if overflow must be handled
	termCols, termRows = getTermSize()
	if termCols != 0 {
		overFlowHandled = true
	}
}

// GetTermSize returns the last known terminal size.
// It is either updated automatically on terminal resize on Unix like systems
// or updated at each refresh/update interval for windows.
func GetTermSize() (cols, rows int) {
	return termCols, termRows
}
