package main

import (
	"errors"
	"fmt"
)

var errIO = errors.New("io")

func run() int {
	a := fmt.Errorf("a: %w", errIO)
	b := errors.New("b")
	joined := errors.Join(a, b)
	if errors.Is(joined, errIO) {
		return 1
	}
	return 0
}
