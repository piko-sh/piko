package main

func fact(n int, acc int) int {
	if n <= 1 {
		return acc
	}
	return fact(n-1, n*acc)
}

func run() int {
	return fact(10, 1)
}
