package main

import (
	"fmt"
	"reflect"
)

type ResultStatus int

const (
	ResultStatusUnknown ResultStatus = iota
	ResultStatusWaiting
	ResultStatusDoing
	ResultStatusDone
)

func (rs ResultStatus) String() string {
	return [...]string{"Unknown", "Waiting", "Doing", "Done"}[rs]
}

func run() string {
	result := ResultStatusWaiting

	if result == ResultStatusWaiting {
		return fmt.Sprintf("Waiting: %s - %s", result.String(), reflect.TypeOf(result).String())
	}

	if result == ResultStatusDoing {
		return fmt.Sprintf("Doing: %s - %s", result.String(), reflect.TypeOf(result).String())
	}

	if result == ResultStatusDone {
		return fmt.Sprintf("Done: %s - %s", result.String(), reflect.TypeOf(result).String())
	}

	if result == ResultStatusUnknown {
		return fmt.Sprintf("Unknown: %s - %s", result.String(), reflect.TypeOf(result).String())
	}

	return fmt.Sprintf("I should not be here at all: %s - %s", result.String(), reflect.TypeOf(result).String())
}
