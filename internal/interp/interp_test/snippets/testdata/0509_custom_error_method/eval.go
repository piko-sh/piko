package main

import "fmt"

type NotFoundErr struct {
	Key string
}

func (e *NotFoundErr) Error() string {
	return fmt.Sprintf("not found: %s", e.Key)
}

func lookup(k string) error {
	return &NotFoundErr{Key: k}
}

func run() int {
	err := lookup("user")
	if err.Error() == "not found: user" {
		return 1
	}
	return 0
}
