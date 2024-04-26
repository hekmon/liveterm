package liveterm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"
)

var (
	waitBuf           bytes.Buffer
	waitCtx           context.Context
	waitCtxCancel     context.CancelFunc
	delayedStopSignal chan struct{}
)

// Bypass creates an io.Writer which allow to write permalent stuff to the terminal while liveterm is running.
// Do not forget to include a final '\n' when writting to it.
func Bypass() io.Writer {
	return bypass{}
}

type bypass struct{}

func (bypass) Write(p []byte) (n int, err error) {
	defer mtx.Unlock()
	mtx.Lock()
	// if liveterm is not started, out is nil
	if termOutput == nil {
		err = errors.New("liveterm is not started, can not write to terminal")
		return
	}
	// check if we are within a wait period
	if overFlowHandled && waitUntil.After(time.Now()) {
		// write it to a temporary buffer
		n, err = waitBuf.Write(p)
		// Start the delayer bypass writer if not already started
		if waitCtx == nil {
			waitCtx, waitCtxCancel = context.WithCancel(context.Background())
			delayedStopSignal = make(chan struct{})
			go delayedBypassWritter()
		}
		return
	}
	// erase current dynamic data
	erase()
	// write permanent data
	if n, err = termOutput.Write(p); err != nil {
		return
	}
	// rewrite the last known dynamic data after it
	_, err = termOutput.Write(lastBuf.Bytes())
	return
}

func delayedBypassWritter() {
	defer close(delayedStopSignal)
loop:
	for {
		mtx.Lock()
		waitTime := time.NewTimer(time.Until(waitUntil))
		mtx.Unlock()
		select {
		case <-waitTime.C:
			mtx.Lock()
			// In case wait duration has been reset during our wait
			if waitUntil.After(time.Now()) {
				mtx.Unlock()
				continue loop
			}
			// Wait time is over, let's by pass
			erase()
			_, _ = termOutput.Write(waitBuf.Bytes())
			_, _ = termOutput.Write(lastBuf.Bytes())
			waitBuf.Reset()
			// Before exiting, mark ourself as not started
			waitCtx = nil
			waitCtxCancel = nil
			// Our work is done
			mtx.Unlock()
			return
		case <-waitCtx.Done():
			// liveterm is being stopped, let's flush the buffer
			// do not try to lock the mutex as it is being locked by the main worker who canceled our context
			erase()
			_, _ = termOutput.Write(waitBuf.Bytes())
			_, _ = termOutput.Write(lastBuf.Bytes())
			waitBuf.Reset()
			// Mark ourself as not started before exiting
			waitCtx = nil
			waitCtxCancel = nil
			return
		}
	}
}
