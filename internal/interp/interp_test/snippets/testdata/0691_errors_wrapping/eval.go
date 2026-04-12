package main

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

type DBError struct {
	Table string
	Cause error
}

func (e *DBError) Error() string {
	return "db[" + e.Table + "]: " + e.Cause.Error()
}

func (e *DBError) Unwrap() error {
	return e.Cause
}

func deepWrap() error {
	first := fmt.Errorf("layer1: %w", ErrNotFound)
	second := fmt.Errorf("layer2: %w", first)
	return &DBError{Table: "users", Cause: second}
}

func run() string {
	result := ""

	root := deepWrap()
	result += "msg:" + root.Error() + ";"

	if errors.Is(root, ErrNotFound) {
		result += "isNotFound:yes;"
	} else {
		result += "isNotFound:no;"
	}

	if errors.Is(root, ErrForbidden) {
		result += "isForbidden:yes;"
	} else {
		result += "isForbidden:no;"
	}

	var dbErr *DBError
	if errors.As(root, &dbErr) {
		result += "asDB:" + dbErr.Table + ";"
	} else {
		result += "asDB:no;"
	}

	unwrapped := errors.Unwrap(root)
	if unwrapped != nil {
		result += "unwrap1:" + unwrapped.Error() + ";"
	}

	joined := errors.Join(ErrNotFound, ErrForbidden)
	if errors.Is(joined, ErrNotFound) && errors.Is(joined, ErrForbidden) {
		result += "joinedHasBoth:yes"
	} else {
		result += "joinedHasBoth:no"
	}

	return result
}
