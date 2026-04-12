package main

import "errors"

type LayeredErr struct {
	Code int
}

func (e *LayeredErr) Error() string {
	return "layered"
}

type wrappingErr struct {
	msg   string
	inner error
}

func (e *wrappingErr) Error() string { return e.msg }
func (e *wrappingErr) Unwrap() error { return e.inner }

func produce() error {
	inner := &LayeredErr{Code: 42}
	return &wrappingErr{msg: "outer", inner: inner}
}

func run() int {
	err := produce()
	var target *LayeredErr
	if errors.As(err, &target) {
		return target.Code
	}
	return -1
}
