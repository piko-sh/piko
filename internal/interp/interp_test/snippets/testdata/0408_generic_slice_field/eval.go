package main

type box[T any] struct {
	items []T
}

func newBox[T any](items []T) box[T] {
	return box[T]{items: items}
}

func run() []int {
	b := newBox([]int{1, 2, 3})
	return b.items
}
