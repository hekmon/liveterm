package liveterm

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

/*
   Erase with escape sequences
*/

/*
Easy to read, but with some mistakes it seems:
https://shiroyasha.svbtle.com/escape-sequences-a-quick-guide-1

More accurate but not complete:
https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences

Complete ?
https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_

Nice one too:
https://learn.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences
*/

// esc is the ASCII code for escape character
const esc = 27

var (
	// Escape sequences
	moveCursorToColumn = func(col int) string {
		return fmt.Sprintf("%c[%dG", esc, col)
	}
	moveCursorUp = func(nbLines int) string {
		return fmt.Sprintf("%c[%dA", esc, nbLines)
	}
	clearLine = fmt.Sprintf("%c[2K", esc)
)

var (
	// Helpers
	cursorBeginningOfLine = moveCursorToColumn(0)
	cursorPreviousLine    = moveCursorUp(1)
	clearCurrentLine      = cursorBeginningOfLine + clearLine
	clearPreviousLine     = cursorPreviousLine + clearLine
)

func terminalCleanUp(linesCount int) {
	// Clear the current line in case the cursor is not at the beginning of the line,
	// for example if SetRawUpdateFx() has been used and no '\n' has been written at the end.
	_, _ = fmt.Fprint(out, clearCurrentLine)
	// clear the rest of the lines
	_, _ = fmt.Fprint(out, strings.Repeat(clearPreviousLine, linesCount))
}
