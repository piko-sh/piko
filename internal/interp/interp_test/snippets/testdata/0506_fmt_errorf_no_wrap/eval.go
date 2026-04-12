package main

import (
	"errors"
	"fmt"
)

var errBase = errors.New("base")

func run() int {
	err := fmt.Errorf("contextual: %v (no wrap)", errBase)
	if errors.Is(err, errBase) {
		return 0
	}
	return 1
}
