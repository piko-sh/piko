package main

type Box[T any] struct {
	Value T
}

func wrap[T any](v T) Box[T] {
	return Box[T]{Value: v}
}

func run() int {
	outer := wrap(wrap(wrap(7)))
	return outer.Value.Value.Value
}
