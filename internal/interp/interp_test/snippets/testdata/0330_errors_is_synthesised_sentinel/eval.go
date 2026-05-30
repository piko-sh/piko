package main

import "errors"

type SentinelErr struct {
	Tag string
}

func (e *SentinelErr) Error() string {
	return "sentinel:" + e.Tag
}

type wrappingErr struct {
	msg   string
	inner error
}

func (e *wrappingErr) Error() string { return e.msg }
func (e *wrappingErr) Unwrap() error { return e.inner }

var ErrNotFound = &SentinelErr{Tag: "not-found"}

func run() int {
	wrapped := &wrappingErr{msg: "layer", inner: ErrNotFound}
	matches := 0
	if errors.Is(ErrNotFound, ErrNotFound) {
		matches += 1
	}
	if errors.Is(wrapped, ErrNotFound) {
		matches += 10
	}
	other := &SentinelErr{Tag: "different"}
	if errors.Is(other, ErrNotFound) {
		matches += 100
	}
	return matches
}
