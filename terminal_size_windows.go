//go:build windows

package liveterm

import (
	"os"
	"unsafe"
)

func getTermSize() (cols, rows int) {
	// Open term
	out, err := os.Open("CONOUT$")
	if err != nil {
		return
	}
	defer out.Close()
	// Get term infos
	var csbi consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(out.Fd(), uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 {
		return
	}
	// Extract term size
	termCols, termRows = int(csbi.window.right-csbi.window.left+1), int(csbi.window.bottom-csbi.window.top+1)
	return
}
