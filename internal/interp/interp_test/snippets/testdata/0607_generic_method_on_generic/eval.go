package main

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

func run() int {
	var s Stack[int]
	s.Push(1)
	s.Push(2)
	s.Push(3)
	return s.Len()
}
