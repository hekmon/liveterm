# liveterm
[![PkgGoDev](https://pkg.go.dev/badge/github.com/hekmon/liveterm)](https://pkg.go.dev/github.com/hekmon/liveterm)

`liveterm` is a go library for updating terminal output in realtime. It is a fork of the really usefull [uilive](https://github.com/gosuri/uilive).

Major differences are:
* Switch from an async push update model to a sync pull update model
* Better handling of cleaning the right amount of lines with terminal resizes
* Helpers to get up to date terminal size to help user formats its data
    * Size is automatically updated if terminal is resized, simply call the helper at the beginning of your formating fx (see the [example](example/main.go))
* Support for incomplete lines
    * User can push raw bytes even if not ended by `\n`
    * Cursor will stay at the end of the line (instead of the beginning of a new line)
    * But the line will be erased properly anyway
* Support for runes (Unicode / UTF-8)
  * A rune can has length (byte representation) different from its column (printing) representation
  * For example a rune with a 3 bytes representation can only use 2 columns on the terminal
  * Computing the actual lines printed to the terminal (especially when the original lines overflow and create new ones) in order to erase them after can not rely on bytes len with unicode
  * So lines lenght are based on unicode rune width instead of byte len in `uilive`

Be sure to check [liveprogress](https://github.com/hekmon/liveprogress) as well !

## Update model

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

### liveterm

With `liveterm` I wanted a more efficient, sync pull based approach:
* You register a function that returns the data you want to be (re)printed
* At each tick, `liveterm` will call that function to get up to date data before printing it
* Between each tick, `liveterm` sleeps and no buffer or mutex are used for nothing

## Usage Example

Full source for the below example is in [example/main.go](example/main.go).

Simplified:

```go
// Change default configuration if needed
liveterm.RefreshInterval = 100 * time.Millisecond
liveterm.Output = os.Stdout

// Set the function that will return the data to be displayed
// This can be done or changed even after Start() has been called
liveterm.SetSingleLineUpdateFx(getStatsFx)

// Start live printing
liveterm.Start()

// [ ... ]

// Let's write something to stdout while liveterm is running
fmt.Fprintf(liveterm.Bypass(), "This is a message that will be displayed on stdout while the counter is running\n")

// [ ... ]

// Release stdout
liveterm.Stop(false)
```

![Example output](example/example.gif)

## Installation

```sh
$ go get -v github.com/hekmon/liveterm/v2
```
