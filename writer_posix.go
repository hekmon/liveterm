//go:build !windows

package termlive

import (
	"fmt"
	"strings"
)

// https://shiroyasha.svbtle.com/escape-sequences-a-quick-guide-1
var (
	cursorUP          = fmt.Sprintf("%c[1A", esc)
	clearLine         = fmt.Sprintf("%c[2K", esc)
	clearPreviousLine = cursorUP + clearLine
)

func clearLines() {
	_, _ = fmt.Fprint(out, strings.Repeat(clearPreviousLine, lineCount))
}
