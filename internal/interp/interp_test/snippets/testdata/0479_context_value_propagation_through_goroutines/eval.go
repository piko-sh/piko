package main

import "context"

type ctxKey string

func run() int {
	ctx := context.WithValue(context.Background(), ctxKey("k"), 42)
	done := make(chan int, 1)
	go func() {
		v := ctx.Value(ctxKey("k"))
		if iv, ok := v.(int); ok {
			done <- iv
			return
		}
		done <- 0
	}()
	return <-done
}
