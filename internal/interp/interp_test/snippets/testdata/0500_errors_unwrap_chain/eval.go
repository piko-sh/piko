package main

import (
	"errors"
	"fmt"
)

func run() int {
	base := errors.New("base")
	wrapped := fmt.Errorf("middle: %w", base)
	outer := fmt.Errorf("outer: %w", wrapped)
	count := 0
	for outer != nil {
		count++
		outer = errors.Unwrap(outer)
	}
	return count
}
