//go:build windows

package liveterm

import (
	"io"
	"syscall"
	"unsafe"

	"github.com/mattn/go-isatty"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procSetConsoleCursorPosition   = kernel32.NewProc("SetConsoleCursorPosition")
	procFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
)

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

// clearLines is unsafe ! It must be called within a mutex lock by one of its callers
func clearLines() {
	fout, ok := out.(fdWriter)
	if !ok || isatty.IsTerminal(fout.Fd()) {
		/*
			Either the output:
			- does not have a file descriptor: we can not test if it is a terminal and
			  windows legacy code cannot be used either, let's try to use terminal escape codes and hope it works
			- has a file descriptor and is a terminal (modern windows, see https://en.wikipedia.org/wiki/ANSI_escape_code#DOS_and_Windows):
			  definitely use terminal escape codes
		*/
		terminalCleanUp()
		return
	}
	// output has a file descriptor but is not a tty: Let's go with legacy windows console manipulation
	fd := fout.Fd()
	csbi := getCSBInfos(fd)
	// Clear the current line in case the cursor is not at the beginning of the line,
	// for example if SetRawUpdateFx() has been used and no '\n' has been written at the end.
	csbi.cursorPosition.x = csbi.window.left
	moveCursorFd(fd, csbi)
	clearLineFd(fd, csbi)
	// clear the rest of the lines
	for i := 0; i < lineCount; i++ {
		// move the cursor up
		csbi.cursorPosition.y--
		moveCursorFd(fd, csbi)
		// clear the line
		clearLineFd(fd, csbi)
	}
}

func getCSBInfos(fd uintptr) (csbi consoleScreenBufferInfo) {
	_, _, _ = procGetConsoleScreenBufferInfo.Call(fd, uintptr(unsafe.Pointer(&csbi)))
	return
}

func moveCursorFd(fd uintptr, csbi consoleScreenBufferInfo) {
	_, _, _ = procSetConsoleCursorPosition.Call(fd, uintptr(*(*int32)(unsafe.Pointer(&csbi.cursorPosition))))
}

func clearLineFd(fd uintptr, csbi consoleScreenBufferInfo) {
	var w dword
	cursor := coord{
		x: csbi.window.left,
		y: csbi.window.top + csbi.cursorPosition.y,
	}
	count := dword(csbi.size.x)
	_, _, _ = procFillConsoleOutputCharacter.Call(fd, uintptr(' '), uintptr(count), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
}
