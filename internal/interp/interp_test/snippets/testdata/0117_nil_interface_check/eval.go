package main

type Doer interface {
	Do() int
}

var d Doer

func run() int {
	r := 0
	if d == nil {
		r += 1
	}
	return r
}
