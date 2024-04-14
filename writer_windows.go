//go:build windows

package liveterm

import (
	"fmt"
	"io"
	"strings"
	"syscall"
	"unsafe"

	"github.com/mattn/go-isatty"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var (
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procSetConsoleCursorPosition   = kernel32.NewProc("SetConsoleCursorPosition")
	procFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
)

var clear = fmt.Sprintf("%c[%dA%c[2K\r", esc, 0, esc)

type short int16
type dword uint32
type word uint16

type coord struct {
	x short
	y short
}

type smallRect struct {
	left   short
	top    short
	right  short
	bottom short
}

type consoleScreenBufferInfo struct {
	size              coord
	cursorPosition    coord
	attributes        word
	window            smallRect
	maximumWindowSize coord
}

// fdWriter is a writer with a file descriptor.
type fdWriter interface {
	io.Writer
	Fd() uintptr
}

func clearLines() {
	f, ok := out.(fdWriter)
	if ok && !isatty.IsTerminal(f.Fd()) {
		ok = false
	}
	if !ok {
		// untested, if you know how to test it on windows, please open an issue
		// this lacks the feature of cleaning a line not ending by '\n' for now
		_, _ = fmt.Fprint(out, strings.Repeat(clear, lineCount))
		return
	}
	// not a tty, do not use terminal escape codes and use windows specific code
	fd := f.Fd()
	csbi := getCSBInfos(fd)
	// Clear the current line in case the cursor is not at the beginning of the line,
	// for example if SetRawUpdateFx() has been used and no '\n' has been written at the end.
	csbi.cursorPosition.x = csbi.window.left
	moveCursor(fd, csbi)
	clearLine(fd, csbi)
	// Clear previous lines
	for i := 0; i < lineCount; i++ {
		// move the cursor up
		csbi.cursorPosition.y--
		moveCursor(fd, csbi)
		// clear the line
		clearLine(fd, csbi)
	}
}

func getCSBInfos(fd uintptr) (csbi consoleScreenBufferInfo) {
	_, _, _ = procGetConsoleScreenBufferInfo.Call(fd, uintptr(unsafe.Pointer(&csbi)))
	return
}

func moveCursor(fd uintptr, csbi consoleScreenBufferInfo) {
	_, _, _ = procSetConsoleCursorPosition.Call(fd, uintptr(*(*int32)(unsafe.Pointer(&csbi.cursorPosition))))
}

func clearLine(fd uintptr, csbi consoleScreenBufferInfo) {
	var w dword
	cursor := coord{
		x: csbi.window.left,
		y: csbi.window.top + csbi.cursorPosition.y,
	}
	count := dword(csbi.size.x)
	_, _, _ = procFillConsoleOutputCharacter.Call(fd, uintptr(' '), uintptr(count), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
}
