package main

import "fmt"

type Atom struct {
	Symbol string
	Number int
}

func (a Atom) GoString() string {
	return fmt.Sprintf("Atom{Symbol:%q, Number:%d}", a.Symbol, a.Number)
}

func run() string {
	a := Atom{Symbol: "H", Number: 1}
	return fmt.Sprintf("hash=%#v;str=%s", a, a.GoString())
}
