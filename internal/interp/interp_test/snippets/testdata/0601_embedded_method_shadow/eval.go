package main

type Inner struct{}

func (Inner) Greet() string { return "from-inner" }

type Outer struct {
	Inner
}

func (Outer) Greet() string { return "from-outer" }

func run() string {
	o := Outer{}
	return o.Greet() + "|" + o.Inner.Greet()
}
