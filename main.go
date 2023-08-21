package main

import (
	"os"
	"os/signal"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan bool, 1)

	runners := create(done, interrupt)
	defer shutdown(runners)

	// TODO: refactor this
	executeCommand(done)

	waitSignal(done, interrupt)
}

func waitSignal(done chan bool, signal chan os.Signal) {
	for {
		select {
		case <-done:
			return
		case <-signal:
			return
		}
	}
}
