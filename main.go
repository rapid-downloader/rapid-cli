package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, []os.Signal{syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGSTOP, os.Interrupt}...)

	cancelCtx, cancel := context.WithCancel(context.Background())
	ctx := context.WithValue(cancelCtx, "cancel", cancel)

	runners := create(ctx)
	defer shutdown(runners)

	executeCommand(ctx)

	for {
		select {
		case <-interrupt:
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
