package main

import (
	"errors"
	"fmt"
)

var (
	errA = errors.New("error A")
	errB = errors.New("error B")
)

func run() int {
	chain := fmt.Errorf("wrap1: %w", fmt.Errorf("wrap2: %w", errA))
	hasA := errors.Is(chain, errA)
	hasB := errors.Is(chain, errB)
	if hasA && !hasB {
		return 1
	}
	return 0
}
