package main

func makeMultiplier(factor int) func(int) int {
	return func(x int) int {
		return x * factor
	}
}

func makeAdder(offset int) func(int) int {
	return func(x int) int {
		return x + offset
	}
}
