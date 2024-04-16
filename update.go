package liveterm

import (
	"bytes"

	"github.com/mattn/go-runewidth"
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
	var currentLineWidth, runeWidth int
	for _, r := range buf.String() {
		if r == '\n' {
			lineCount++
			currentLineWidth = 0
			continue
		}
		if overFlowHandled {
			runeWidth = runewidth.RuneWidth(r)
			currentLineWidth += runeWidth
			if currentLineWidth > termCols {
				lineCount++
				currentLineWidth = runeWidth
			}
		}
	}
	// Write the current state
	return out.Write(buf.Bytes())
}
