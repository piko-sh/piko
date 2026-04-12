package main

func run() int {
	m := map[int]int{0: 10, 1: 20, 2: 30}
	fns := make([]func() int, 3)
	for k, v := range m {
		v := v
		fns[k] = func() int { return v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}
