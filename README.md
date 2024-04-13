# termlive
[![PkgGoDev](https://pkg.go.dev/badge/github.com/hekmon/termlive)](https://pkg.go.dev/github.com/hekmon/termlive)

`termlive` is a go library for updating terminal output in realtime. It is a fork of the really usefull [uilive](github.com/gosuri/uilive).

## Fork difference

### uilive

`uilive` works with an async push based approach:
* You write (aka "push") to an buffer within `uilive` writer
* Writing to this buffer triggers a mutex lock/unlock
* When ticks kick in, `uilive` read its internal buffer and update the terminal with it

This can cause performance issue when your data change very frequently:
* Let's say you update the buffer 1,000,000 times per second
* But your ui update frequency is 100 milliseconds
* Between each ui update, the internal buffer is modified 100,000 times, 99,999 for nothing
* This is wasted ressources and can cause slowdown because of the mutex constant locking/unlocking

You could throttle you data update with your own ticker but you will end up with 2 tickers on both side, not in sync. Why not use only one ?

### termlive

With `termlive` I wanted a more efficient, sync pull based approach:
* You register a function that returns the data you want to be (re)printed
* At each tick, `termlive` will call that function to get up to date data before printing it
* Between each tick, `termlive` sleeps and no buffer or mutex are used for nothing

## Usage Example

Full source for the below example is in [example/main.go](example/main.go).

```go
// Change default configuration if needed
termlive.RefreshInterval = 100 * time.Millisecond
termlive.UseStdErr = false

// Set the function that will return the data to be displayed
// This can be done or changed even after Start() has been called
termlive.SetSingleLineUpdateFx(getStatsFx)

// Start live printing
termlive.Start()

// [ ... ]

// Let's write something to stdout while termlive is running
fmt.Fprintf(termlive.Bypass(), "This is a message that will be displayed on stdout while the counter is running\n")

// [ ... ]

// Release stdout
termlive.Stop(false)
```

![Example output](example/example.gif)

## Installation

```sh
$ go get -v github.com/hekmon/termlive
```
