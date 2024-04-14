//go:build !windows

package termlive

import (
	"fmt"
	"strings"
)

// https://shiroyasha.svbtle.com/escape-sequences-a-quick-guide-1
var (
	clearLine             = fmt.Sprintf("%c[2K", esc)
	cursorBeginningOfLine = fmt.Sprintf("%c[0E", esc)
	cursorUP1line         = fmt.Sprintf("%c[%dA", esc, 1)
	clearCurrentLine      = cursorBeginningOfLine + clearLine
	clearPreviousLine     = cursorUP1line + clearLine
)

func clearLines() {
	// Clear the current line in case the cursor is not at the beginning of the line,
	// for example if SetRawUpdateFx() has been used and no '\n' has been written at the end.
	_, _ = fmt.Fprint(out, clearCurrentLine)
	// clear the rest of the lines
	_, _ = fmt.Fprint(out, strings.Repeat(clearPreviousLine, lineCount))
}
