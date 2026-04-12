package main

type M[K comparable, V any] = map[K]V

func run() int {
	m := M[string, int]{"a": 1, "b": 2}
	return m["a"] + m["b"]
}
