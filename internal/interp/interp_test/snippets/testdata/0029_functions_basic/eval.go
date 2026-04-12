package main

func double(x int) int {
	return x * 2
}
func triple(x int) int {
	return x * 3
}

func run() int {
	return double(triple(5))
}
