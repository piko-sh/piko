package main

import (
	"context"
	"errors"
	"time"
)

func run() int {
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		50*time.Millisecond,
		errors.New("test timeout"),
	)
	defer cancel()
	done := make(chan int, 1)
	go func() {
		<-ctx.Done()
		done <- 1
	}()
	return <-done
}
