package main

func countdown(n int) int {
	if n <= 0 {
		return 0
	}
	return countdown(n - 1)
}

func run() int {
	return countdown(100000)
}
