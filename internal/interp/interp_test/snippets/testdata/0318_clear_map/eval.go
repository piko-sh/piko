package main

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	clear(m)
	m["x"] = 99
	return len(m) + m["x"]
}
