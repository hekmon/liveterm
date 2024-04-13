//go:build !windows

package termlive

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

var term *os.File
var err error
var sz windowSize

func getTermSize() (int, int) {
	if runtime.GOOS == "openbsd" {
		term, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return 0, 0
		}

	} else {
		term, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			return 0, 0
		}
	}
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
		term.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows)
}
