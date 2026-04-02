package main

type Base struct{ V int }
type Derived struct {
	Base
	V int
}

func run() int {
	d := Derived{Base: Base{V: 1}, V: 2}
	return d.V + d.Base.V
}
