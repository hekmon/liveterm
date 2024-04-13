//go:build !windows

package termlive

import (
	"fmt"
	"strings"
)

// clear the line and move the cursor up
var clear = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

func clearLines() {
	_, _ = fmt.Fprint(out, strings.Repeat(clear, lineCount))
}
