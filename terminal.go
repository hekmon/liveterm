package liveterm

import (
	"errors"
	"fmt"
	"os"
)

var (
	terminalFile *os.File
)

func initTermInfos() (err error) {
	if err = openTerminal(); err != nil {
		return fmt.Errorf("failed to open terminal: %w", err)
	}
	if err = initTermOS(); err != nil {
		return fmt.Errorf("failed to init terminal OS related options: %w", err)
	}
	// If it is, recover its current size
	initTermSize()
	return
}

func clearTermInfos() (err error) {
	if terminalFile == nil {
		return errors.New("terminal connection is already closed")
	}
	err = terminalFile.Close()
	terminalFile = nil
	clearTermOS()
	clearTermSize()
	return
}

func terminalCleanUp(linesCount int) {
	termOutput.ClearLine()
	termOutput.ClearLines(linesCount)
}

/*
   Size related
*/

var (
	termCols, termColsPrevious int
	termRows, termRowsPrevious int
	overFlowHandled            bool
)

func initTermSize() {
	termCols, termRows = getTermSize()
	termColsPrevious, termRowsPrevious = termCols, termRows
	if termCols != 0 {
		overFlowHandled = true
	}
}

func clearTermSize() {
	termCols, termRows = 0, 0
	termColsPrevious, termRowsPrevious = 0, 0
	overFlowHandled = false
}
