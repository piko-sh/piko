package main

func run() int {
	var f func(int) int
	f = func(n int) int {
		if n <= 1 {
			return 1
		}
		return n * f(n-1)
	}
	return f(5)
}
