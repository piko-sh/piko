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

func run() string {
	result := ResultStatusWaiting

	if result == ResultStatusWaiting {
		return fmt.Sprintf("Waiting: %d - %s", result, reflect.TypeOf(result).String())
	}

	if result == ResultStatusDoing {
		return fmt.Sprintf("Doing: %d - %s", result, reflect.TypeOf(result).String())
	}

	if result == ResultStatusDone {
		return fmt.Sprintf("Done: %d - %s", result, reflect.TypeOf(result).String())
	}

	if result == ResultStatusUnknown {
		return fmt.Sprintf("Unknown: %d - %s", result, reflect.TypeOf(result).String())
	}

	return fmt.Sprintf("I should not be here at all: %d - %s", result, reflect.TypeOf(result).String())
}
