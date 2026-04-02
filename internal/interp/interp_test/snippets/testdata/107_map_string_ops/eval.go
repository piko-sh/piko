package main

func run() int {
	m := map[string]int{}
	m["a"] = 10
	m["b"] = 20
	m["c"] = 30
	v, ok := m["b"]
	r := v
	if ok {
		r += 100
	}
	delete(m, "a")
	r += len(m) * 1000
	return r
}
