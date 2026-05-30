package main

import "errors"

var (
	e1 = errors.New("one")
	e2 = errors.New("two")
	e3 = errors.New("three")
)

func run() int {
	joined := errors.Join(e1, e2, e3)
	if errors.Is(joined, e1) && errors.Is(joined, e2) && errors.Is(joined, e3) {
		return 1
	}
	return 0
}
