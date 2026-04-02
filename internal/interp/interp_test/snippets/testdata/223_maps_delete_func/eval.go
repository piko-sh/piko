package main

import "maps"

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	maps.DeleteFunc(m, func(k string, v int) bool { return v%2 == 0 })
	result := 0
	for _, v := range m {
		result += v
	}
	return result
}
