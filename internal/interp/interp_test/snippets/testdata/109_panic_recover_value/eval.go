package main

func safeDivide(a, b int) int {
	defer func() {
		recover()
	}()
	return a / b
}

func run() int {
	return safeDivide(10, 2)
}
