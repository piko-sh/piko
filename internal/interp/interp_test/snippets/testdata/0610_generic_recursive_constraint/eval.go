package main

func equal[T comparable](a, b T) bool {
	return a == b
}

func run() int {
	if equal(1, 1) && equal("x", "x") && !equal(2, 3) {
		return 1
	}
	return 0
}
