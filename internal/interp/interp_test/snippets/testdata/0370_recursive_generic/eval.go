package main

func sumTo[T int | int64](n T) T {
	if n <= 0 {
		return 0
	}
	return n + sumTo(n-1)
}

func run() int {
	return int(sumTo(10))
}
