package liveterm

var (
	termCols, termColsPrevious int
	termRows, termRowsPrevious int
	overFlowHandled            bool
)

func initTermSize() {
	termCols, termRows = getTermSize()
	if termCols != 0 {
		overFlowHandled = true
	}
}

func clearTermSize() {
	termCols, termRows, termColsPrevious, termRowsPrevious = 0, 0, 0, 0
	overFlowHandled = false
}
