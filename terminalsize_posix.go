//go:build !windows

package liveterm

import (
	"syscall"
	"unsafe"
)

type windowSize struct {
	rows uint16
	cols uint16
}

func getTermSize() (cols, rows int) {
	if terminalFile == nil {
		return
	}
	var sz windowSize
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, terminalFile.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows)
}
