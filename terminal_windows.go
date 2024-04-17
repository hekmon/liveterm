//go:build windows

package liveterm

import (
	"errors"
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/mattn/go-isatty"
)

var (
	outFd         uintptr
	outIsTerminal bool
)

func openTerminal() (err error) {
	terminalFile, err = os.Open("CONOUT$")
	return
}

// fdWriter is a writer with a file descriptor.
type fdWriter interface {
	io.Writer
	Fd() uintptr
}

func initTermOS() (err error) {
	fout, ok := out.(fdWriter)
	if !ok {
		return errors.New("can not extract file descriptor from output writer")
	}
	outFd = fout.Fd()
	outIsTerminal = isatty.IsTerminal(outFd)
	return
}

func clearTermOS() {
	outFd = 0
	outIsTerminal = false
}

/*
	Size related
*/

func getTermSize() (cols, rows int) {
	// Get term infos
	var csbi consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(terminalFile.Fd(), uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 {
		return
	}
	// Extract term size
	return int(csbi.window.right - csbi.window.left + 1), int(csbi.window.bottom - csbi.window.top + 1)
}

/*
	Lines clearing related
*/

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

// clearLines is unsafe ! It must be called within a mutex lock by one of its callers
func clearLines(linesCount int) {
	if outIsTerminal {
		terminalCleanUp(linesCount)
		return
	}
	// output is not a tty: Let's go with legacy windows console manipulation
	csbi := getCSBInfos(outFd)
	// Clear the current line in case the cursor is not at the beginning of the line,
	// for example if SetRawUpdateFx() has been used and no '\n' has been written at the end.
	csbi.cursorPosition.x = csbi.window.left
	moveCursorFd(outFd, csbi)
	clearLineFd(outFd, csbi)
	// clear the rest of the lines
	for i := 0; i < linesCount; i++ {
		// move the cursor up
		csbi.cursorPosition.y--
		moveCursorFd(outFd, csbi)
		// clear the line
		clearLineFd(outFd, csbi)
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
