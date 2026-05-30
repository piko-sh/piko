package main

import (
	"context"
	"errors"
)

func run() int {
	ctx, cancel := context.WithCancelCause(context.Background())
	done := make(chan int, 1)
	go func() {
		<-ctx.Done()
		done <- 1
	}()
	cancel(errors.New("test cancel"))
	return <-done
}
