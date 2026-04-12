package main

func double(x int) int {
	return x * 2
}

func run() int {
	f := double
	return f(21)
}
