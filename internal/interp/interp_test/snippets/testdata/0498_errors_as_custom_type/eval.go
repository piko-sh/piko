package main

import (
	"errors"
	"fmt"
)

type MyErr struct {
	Code int
}

func (e *MyErr) Error() string {
	return fmt.Sprintf("my-err code=%d", e.Code)
}

func run() int {
	base := &MyErr{Code: 7}
	wrapped := fmt.Errorf("ctx: %w", base)
	var target *MyErr
	if errors.As(wrapped, &target) {
		return target.Code
	}
	return 0
}
