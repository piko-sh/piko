package main

type Box[T any] struct {
	Value T
}

func (b Box[T]) Get() T {
	return b.Value
}
func (b *Box[T]) Set(v T) {
	b.Value = v
}

func run() int {
	b := &Box[int]{Value: 10}
	b.Set(42)
	return b.Get()
}
