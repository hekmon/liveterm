//go:build windows

package liveterm

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type (
	short int16
	word  uint16
)

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

func getTermSize() (cols, rows int) {
	if terminalFile == nil {
		return
	}
	// Get term infos
	var csbi consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(terminalFile.Fd(), uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 {
		return
	}
	// Extract term size
	return int(csbi.window.right - csbi.window.left + 1), int(csbi.window.bottom - csbi.window.top + 1)
}
