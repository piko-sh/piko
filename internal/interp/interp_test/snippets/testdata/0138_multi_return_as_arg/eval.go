package main

func pair() (int, int) {
	return 10, 20
}
func sum(a, b int) int {
	return a + b
}

func run() int {
	return sum(pair())
}
