package main

import "fmt"

type TestingT interface {
	Errorf(format string, args ...interface{})
}

type localT struct {
	failures []string
}

func (t *localT) Errorf(format string, args ...interface{}) {
	t.failures = append(t.failures, fmt.Sprintf(format, args...))
}

func Fail(t TestingT, failureMessage string) bool {
	t.Errorf("\n%s", failureMessage)
	return false
}

func run() int {
	t := &localT{}
	_ = Fail(t, "msg")
	return len(t.failures)
}
