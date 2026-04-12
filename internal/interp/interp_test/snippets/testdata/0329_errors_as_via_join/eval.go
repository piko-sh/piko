package main

import "errors"

type JoinedErr struct {
	Code int
}

func (e *JoinedErr) Error() string {
	return "joined"
}

func produce() error {
	return errors.Join(errors.New("alpha"), &JoinedErr{Code: 99}, errors.New("gamma"))
}

func run() int {
	err := produce()
	var target *JoinedErr
	if errors.As(err, &target) {
		return target.Code
	}
	return -1
}
