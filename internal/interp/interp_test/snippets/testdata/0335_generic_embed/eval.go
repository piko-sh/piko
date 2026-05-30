package main

type Base struct {
	Tag string
}

func (b Base) Label() string {
	return "[" + b.Tag + "]"
}

type Container[T any] struct {
	Base
	Item T
}

func run() string {
	c := Container[int]{Base: Base{Tag: "x"}, Item: 9}
	return c.Label()
}
