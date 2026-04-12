package main

func makeMap[K comparable, V any](pairs []K, values []V) map[K]V {
	m := make(map[K]V, len(pairs))
	for i, k := range pairs {
		m[k] = values[i]
	}
	return m
}

func run() int {
	m := makeMap([]string{"a", "b", "c"}, []int{1, 2, 3})
	return m["a"] + m["b"] + m["c"]
}
