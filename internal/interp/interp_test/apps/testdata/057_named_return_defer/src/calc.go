package main

import "errors"

func computeWithAdjustment(input int) (result int, err error) {
	defer func() {
		adjustResult(&result, 100)
	}()
	if input < 0 {
		err = errors.New("negative")
		err = errors.New(wrapErrorMessage(err.Error(), "calc"))
		return
	}
	result = input * 2
	return
}
