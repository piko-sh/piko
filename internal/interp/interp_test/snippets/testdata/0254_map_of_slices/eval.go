package main

func run() int {
	m := map[string][]int{
		"a": {1, 2, 3},
		"b": {4, 5},
	}
	m["a"] = append(m["a"], 4)
	return len(m["a"]) + m["b"][1]
}
