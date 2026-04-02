package main

func run() int {
	fns := make([]func() int, 3)
	for i, v := range []int{10, 20, 30} {
		fns[i] = func() int { return i*100 + v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}
