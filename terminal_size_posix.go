//go:build !windows

package liveterm

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

type windowSize struct {
	rows uint16
	cols uint16
}

func getTermSize() (cols, rows int) {
	var (
		term *os.File
		err  error
	)
	if runtime.GOOS == "openbsd" {
		term, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return
		}
		defer term.Close()
	} else {
		term, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer term.Close()
	}
	var sz windowSize
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, term.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows)
}
