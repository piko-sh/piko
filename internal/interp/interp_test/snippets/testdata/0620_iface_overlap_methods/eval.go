package main

type A interface {
	Foo() string
}

type B interface {
	Foo() string
	Bar() string
}

type t struct{}

func (t) Foo() string { return "foo" }
func (t) Bar() string { return "bar" }

func run() string {
	var b B = t{}
	var a A = b
	return a.Foo() + "|" + b.Bar()
}
