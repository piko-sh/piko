package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("inner")

func wrapError() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wrapped: %w", err)
		}
	}()
	err = sentinel
	return
}

func setNamedReturn() (n int, msg string) {
	defer func() {
		n = n * 10
		msg = msg + "!"
	}()
	n = 7
	msg = "hi"
	return
}

func mutateRecovered() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	panic("boom")
}

func run() string {
	result := ""

	w := wrapError()
	result += fmt.Sprintf("wrap=%v,is=%t;", w, errors.Is(w, sentinel))

	n, m := setNamedReturn()
	result += fmt.Sprintf("named=%d/%s;", n, m)

	r := mutateRecovered()
	result += fmt.Sprintf("recovered=%v", r)

	return result
}
