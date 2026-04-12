package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func run() string {
	deadline := time.Now().Add(30 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	time.Sleep(60 * time.Millisecond)

	err := ctx.Err()
	return fmt.Sprintf("err_is_deadline=%t;err_nil=%t",
		errors.Is(err, context.DeadlineExceeded),
		err == nil)
}
