package main

func check() int {
	m := map[string]int{"a": 10, "b": 20}
	v, ok := m["a"]
	if ok {
		return v
	}
	return -1
}

func run() int {
	return check()
}
