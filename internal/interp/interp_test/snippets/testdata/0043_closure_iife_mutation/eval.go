package main

func run() int {
	x := 10
	func() {
		x = x + 5
	}()
	return x
}
