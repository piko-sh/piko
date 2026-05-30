package main

func f() (x int) {
	x = 42
	return
}

func run() int {
	return f()
}
