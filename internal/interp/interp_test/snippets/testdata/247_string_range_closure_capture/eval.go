package main

func run() int {
	fns := make([]func() int, 3)
	idx := 0
	for _, r := range "abc" {
		r := r
		i := idx
		fns[i] = func() int { return int(r) }
		idx++
	}
	return fns[0]() + fns[1]() + fns[2]()
}
