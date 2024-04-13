//go:build !windows

package termlive

import (
	"fmt"
	"strings"
)

// clear the line and move the cursor up
var clear = fmt.Sprintf("%c[%dA%c[2K", esc, 1, esc)

func clearLines() {
	_, _ = fmt.Fprint(out, strings.Repeat(clear, lineCount))
}
