package main

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	for k := range m {
		m[k] *= 10
	}
	return m["a"] + m["b"] + m["c"]
}
