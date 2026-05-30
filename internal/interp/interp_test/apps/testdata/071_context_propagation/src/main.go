package main

import (
	"context"
	"errors"
	"fmt"

	"example.com/ctxprop/inner"
)

func entrypoint() string {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test cancel"))
	if inner.Observed(ctx) {
		return "cancelled"
	}
	return "not-cancelled"
}

func main() {
	fmt.Println(entrypoint())
}
