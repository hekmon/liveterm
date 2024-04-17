//go:build !windows

package liveterm

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

func openTerminal() (err error) {
	if runtime.GOOS == "openbsd" {
		terminalFile, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
	} else {
		terminalFile, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	}
	return
}

func initTermOS() (err error) {
	// dummy on Posix
	return
}

func clearTermOS() {
	// dummy on Posix
}

/*
	Size related
*/

type windowSize struct {
	rows uint16
	cols uint16
}

func getTermSize() (cols, rows int) {
	var sz windowSize
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, terminalFile.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows)
}

/*
	Lines clearing related
*/

// clearLines is unsafe ! It must be called within a mutex lock by one of its callers
func clearLines(linesCount int) {
	terminalCleanUp(linesCount)
}
