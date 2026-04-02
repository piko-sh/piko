package main

func run() int {
	base := 100
	compute := func(x int, y int) int {
		return base + x*y
	}
	return compute(3, 4)
}
