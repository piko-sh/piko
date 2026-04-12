package main

func run() int {
	fns := make([]func() int, 3)
	for i := 0; i < 3; i++ {
		fns[i] = func() int { return i }
	}
	return fns[0]() + fns[1]() + fns[2]()
}
