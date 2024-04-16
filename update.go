package liveterm

import (
	"bytes"
	"time"

	"github.com/mattn/go-runewidth"
)

const (
	resizeWait = 500 * time.Millisecond
)

var (
	buf, lastBuf bytes.Buffer
	waitUntil    time.Time
	lineCount    int
)

// update is unsafe ! It must be called within a mutex lock by one of its callers
func update() {
	if (getterLines == nil && getterLine == nil && getterRaw == nil) || out == nil {
		return
	}
	// Update terminal size for erase
	if overFlowHandled {
		termColsPrevious, termRowsPrevious = termCols, termRows
		termCols, termRows = getTermSize()
		if termCols != termColsPrevious || termRows != termRowsPrevious {
			// term has been resized, wait for stability before computing the lines to erase
			// in case the terminal resizing is not done yet
			waitUntil = time.Now().Add(resizeWait)
			return
		}
		// Size is stable between 2 ticks, but are we in a wait period ?
		// Ensure fast rize is handled correctly
		if waitUntil.After(time.Now()) {
			return
		}
	}
	// Build unused buffer with fresh data
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
	// Cleanup terminal based on previous data and current terminal size
	erase()
	// Update terminal with it
	_, _ = out.Write(buf.Bytes())
	// Swap buffers to minimze memory allocation
	lastBuf, buf = buf, lastBuf
}

// erase is unsafe ! It must be called within a mutex lock by one of its callers
func erase() {
	lineCount = 0
	var currentLineWidth, runeWidth int
	for _, r := range lastBuf.String() {
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
	clearLines()
}
