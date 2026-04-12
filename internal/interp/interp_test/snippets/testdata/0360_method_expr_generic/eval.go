package main

type Holder[T any] struct {
	V T
}

func (h Holder[T]) Read() T {
	return h.V
}

func run() int {
	h := Holder[int]{V: 42}
	read := Holder[int].Read
	return read(h)
}
