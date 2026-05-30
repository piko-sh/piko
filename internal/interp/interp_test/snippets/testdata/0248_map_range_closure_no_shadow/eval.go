package main

func run() int {
	m := map[int]int{0: 100, 1: 200, 2: 300}
	fns := make([]func() int, 3)
	for k, v := range m {
		fns[k] = func() int { return v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}
