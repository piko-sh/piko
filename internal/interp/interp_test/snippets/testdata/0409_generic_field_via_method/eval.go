package main

type box[T any] struct {
	value T
}

func (b box[T]) get() T {
	return b.value
}

func run() int {
	b := box[int]{value: 7}
	return b.get()
}
