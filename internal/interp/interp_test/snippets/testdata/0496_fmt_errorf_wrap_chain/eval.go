package main

import (
	"errors"
	"fmt"
)

var errBase = errors.New("base failure")

func run() int {
	wrapped := fmt.Errorf("layer A: %w", errBase)
	wrapped2 := fmt.Errorf("layer B: %w", wrapped)
	if errors.Is(wrapped2, errBase) {
		return 1
	}
	return 0
}
