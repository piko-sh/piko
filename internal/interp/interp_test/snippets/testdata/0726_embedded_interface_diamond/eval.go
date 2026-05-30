package main

import "fmt"

type A interface {
	M() string
}

type B interface {
	M() string
}

type Diamond interface {
	A
	B
}

type Impl struct{}

func (Impl) M() string { return "impl" }

func run() string {
	result := ""
	var d Diamond = Impl{}
	result += fmt.Sprintf("d=%s;", d.M())

	var a A = d
	result += fmt.Sprintf("a=%s;", a.M())

	var b B = d
	result += fmt.Sprintf("b=%s", b.M())
	return result
}
