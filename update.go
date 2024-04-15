package liveterm

import (
	"bytes"
)

var (
	buf       bytes.Buffer
	lineCount int
)

// update is unsafe ! It must be called within a mutex lock by one of its callers
func update() {
	if (getterLines == nil && getterLine == nil && getterRaw == nil) || out == nil {
		return
	}
	// Update terminal size manually if necessary before calling data fx (which may rely on termSize)
	if overFlowHandled && !termSizeAutoUpdate {
		termCols, termRows = getTermSize()
	}
	// Rebuild buffer with fresh data
	buf.Reset()
	switch {
	case getterLines != nil:
		for _, line := range getterLines() {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	case getterLine != nil:
		buf.WriteString(getterLine())
		buf.WriteByte('\n')
	case getterRaw != nil:
		buf.Write(getterRaw())
	}
	// Update terminal with it
	erase()
	_, _ = write()
}

// erase is unsafe ! It must be called within a mutex lock by one of its callers
func erase() {
	clearLines()
	lineCount = 0
}

// write is unsafe ! It must be called within a mutex lock by one of its callers
func write() (n int, err error) {
	// Count the number of actual term lines we are about to write for futur clearLines() call
	var currentLine bytes.Buffer
	for _, b := range buf.Bytes() {
		if b == '\n' {
			lineCount++
			currentLine.Reset()
		} else if overFlowHandled {
			currentLine.Write([]byte{b})
			if currentLine.Len() > termCols {
				lineCount++
				currentLine.Reset()
			}
		}
	}
	// Write the current state
	return out.Write(buf.Bytes())
}
