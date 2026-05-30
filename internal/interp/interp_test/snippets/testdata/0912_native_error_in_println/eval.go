package main

import "fmt"

type badInput struct {
	code int
}

func (b badInput) Error() string {
	return fmt.Sprintf("bad input (code=%d)", b.code)
}

func run() string {
	var err error = badInput{code: 99}
	return fmt.Sprintf("error: %s", err)
}
