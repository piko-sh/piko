package main

type Inner struct {
	A int
	B int
}

type Outer struct {
	Label string
	Inner Inner
}

func run() int {
	o := Outer{Label: "x", Inner: Inner{A: 7, B: 8}}
	o.Inner.A *= 10
	return o.Inner.A + o.Inner.B
}
