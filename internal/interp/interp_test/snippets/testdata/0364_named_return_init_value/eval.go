package main

func compute() (n int) {
	n = 10
	defer func() {
		n *= 3
	}()
	n = 5
	return
}

func run() int {
	return compute()
}
