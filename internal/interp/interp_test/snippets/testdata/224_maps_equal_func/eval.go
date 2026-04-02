package main

import "maps"

func run() bool {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"a": 10, "b": 20}
	return maps.EqualFunc(m1, m2, func(v1, v2 int) bool { return v1*10 == v2 })
}
