package main

import "errors"

func run() int {
	a := errors.New("same text")
	b := errors.New("same text")
	if a == b {
		return 0
	}
	return 1
}
