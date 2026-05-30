package main

import "errors"

type CodedErr struct {
	Code int
}

func (e *CodedErr) Error() string {
	return "coded"
}

type Coded interface {
	error
	codeMarker()
}

func (*CodedErr) codeMarker() {}

func produce() error {
	return &CodedErr{Code: 7}
}

func run() int {
	err := produce()
	var target Coded
	if errors.As(err, &target) {
		if c, ok := target.(*CodedErr); ok {
			return c.Code
		}
	}
	return -1
}
