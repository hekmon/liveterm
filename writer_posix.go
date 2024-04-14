//go:build !windows

package termlive

import (
	"fmt"
	"strings"
)

// https://shiroyasha.svbtle.com/escape-sequences-a-quick-guide-1
var (
	cursorUP1line     = fmt.Sprintf("%c[%dA", esc, 1)
	clearLine         = fmt.Sprintf("%c[2K", esc)
	clearPreviousLine = cursorUP1line + clearLine
)

func clearLines() {
	_, _ = fmt.Fprint(out, strings.Repeat(clearPreviousLine, lineCount))
}
