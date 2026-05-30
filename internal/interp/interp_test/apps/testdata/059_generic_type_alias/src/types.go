package main

type Box[T any] struct {
	Value T
}

type IntBox = Box[int]

type Pair[K comparable, V any] = struct {
	Key   K
	Value V
}
