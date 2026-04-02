package main

func add(a, b int) int { return a + b }
func init() {
	_ = add(20, 22)
}

func run() int {
	return add(1, 2)
}
