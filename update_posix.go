//go:build !windows

package liveterm

// clearLines is unsafe ! It must be called within a mutex lock by one of its callers
func clearLines(linesCount int) {
	terminalCleanUp(linesCount)
}
