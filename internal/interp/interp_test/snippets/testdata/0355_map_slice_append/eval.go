package main

func run() int {
	m := map[string][]int{}
	m["a"] = append(m["a"], 1, 2)
	m["a"] = append(m["a"], 3)
	m["b"] = append(m["b"], 10)
	total := 0
	for _, v := range m["a"] {
		total += v
	}
	for _, v := range m["b"] {
		total += v
	}
	return total
}
