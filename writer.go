package liveterm

import (
	"bytes"
	"errors"
	"io"
)

// ESC is the ASCII code for escape character
const esc = 27

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
		termSize = getTermSize()
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
			if currentLine.Len() > termSize.Cols {
				lineCount++
				currentLine.Reset()
			}
		}
	}
	// Write the current state
	return out.Write(buf.Bytes())
}

// Bypass creates an io.Writer which allows to write a permalent lines to the terminal. Do not forget to include a final '\n' when writting to it.
func Bypass() io.Writer {
	return &bypass{}
}

type bypass struct{}

func (bypass) Write(p []byte) (n int, err error) {
	defer mtx.Unlock()
	mtx.Lock()
	// if liveterm is not started, out is nil
	if out == nil {
		err = errors.New("liveterm is not started, can not write to terminal")
		return
	}
	// erase current dynamic data
	erase()
	// write permanent data
	if n, err = out.Write(p); err != nil {
		return
	}
	// rewrite the last known dynamic data after it
	_, err = write()
	return
}
