package main

import (
	"context"
	"errors"
	"time"
)

func run() int {
	root, rootCancel := context.WithCancelCause(context.Background())
	defer rootCancel(nil)
	mid, midCancel := context.WithTimeoutCause(root, 100*time.Millisecond, errors.New("mid timeout"))
	defer midCancel()
	leaf := context.WithValue(mid, "key", 42)

	done := make(chan int, 1)
	go func() {
		<-leaf.Done()
		done <- 1
	}()
	rootCancel(errors.New("root"))
	return <-done
}
