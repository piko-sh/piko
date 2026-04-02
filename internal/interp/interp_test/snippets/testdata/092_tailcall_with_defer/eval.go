package main

func factDefer(n int, acc int) int {
	defer func() {}()
	if n <= 1 {
		return acc
	}
	return factDefer(n-1, n*acc)
}

func run() int {
	return factDefer(10, 1)
}
