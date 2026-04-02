package main

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	delete(m, "b")
	return len(m)
}
