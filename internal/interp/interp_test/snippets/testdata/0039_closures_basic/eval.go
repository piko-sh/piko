package main

func run() int {
	factor := 5
	mul := func(v int) int {
		return v * factor
	}
	return mul(3) + mul(7)
}
