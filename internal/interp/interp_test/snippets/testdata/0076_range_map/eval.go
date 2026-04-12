package main

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	sum := 0
	for _, v := range m {
		sum = sum + v
	}
	return sum
}
