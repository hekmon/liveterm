package liveterm

import (
	"fmt"
	"strings"
)

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

// terminalCleanUp is unsafe ! It must be called within a mutex lock by one of its callers
func terminalCleanUp() {
	// Clear the current line in case the cursor is not at the beginning of the line,
	// for example if SetRawUpdateFx() has been used and no '\n' has been written at the end.
	_, _ = fmt.Fprint(out, clearCurrentLine)
	// clear the rest of the lines
	_, _ = fmt.Fprint(out, strings.Repeat(clearPreviousLine, lineCount))
}
