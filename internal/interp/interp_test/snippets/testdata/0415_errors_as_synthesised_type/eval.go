package main

import "errors"

type MyErr struct {
	Code int
}

func (e *MyErr) Error() string {
	return "myerr"
}

func wrap() error {
	return &MyErr{Code: 7}
}

func run() int {
	err := wrap()
	var target *MyErr
	if errors.As(err, &target) {
		return target.Code
	}
	return -1
}
