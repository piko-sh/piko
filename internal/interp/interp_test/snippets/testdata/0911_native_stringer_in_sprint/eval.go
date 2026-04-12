package main

import "fmt"

type Label string

func (l Label) String() string {
	return "<" + string(l) + ">"
}

func run() string {
	a := Label("alpha")
	b := Label("beta")
	return fmt.Sprint(a, "+", b)
}
