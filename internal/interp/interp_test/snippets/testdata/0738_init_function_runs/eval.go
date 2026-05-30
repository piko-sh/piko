package main

import "fmt"

var initLog []string

func init() {
	initLog = append(initLog, "first")
}

func init() {
	initLog = append(initLog, "second")
}

func init() {
	defer func() {
		if r := recover(); r != nil {
			initLog = append(initLog, fmt.Sprintf("recovered=%v", r))
		}
	}()
	panic("init-panic")
}

func init() {
	initLog = append(initLog, "after-recovery")
}

func run() string {
	result := fmt.Sprintf("count=%d;", len(initLog))
	for _, entry := range initLog {
		result += entry + ";"
	}
	return result
}
