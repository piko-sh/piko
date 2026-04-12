package main

type Stack[T any] struct {
	values []T
}

func newStack[T any]() *Stack[T] {
	return &Stack[T]{}
}
