package main

func run() int {
	x := 1
	defer func() {
		_ = x
	}()
	x = 100
	return x
}
