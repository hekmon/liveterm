//go:build !windows

package liveterm

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"
)

type windowSize struct {
	rows uint16
	cols uint16
}

func getTermSize() (ts TermSize) {
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
	ts.Cols, ts.Rows = int(sz.cols), int(sz.rows)
	return
}

// startListeningForTermResize is unsafe ! It must be called within a mutex lock by one of its callers
func startListeningForTermResize() {
	termSizeChan = make(chan os.Signal, 1)
	signal.Notify(termSizeChan, syscall.SIGWINCH)
	termSizeAutoUpdate = true
}

// stopListeningForTermResize is unsafe ! It must be called within a mutex lock by one of its callers
func stopListeningForTermResize() {
	termSizeAutoUpdate = false
	signal.Stop(termSizeChan)
	termSizeChan = nil
}
