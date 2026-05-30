package main

func run() int {
	x := 10
	f := func() int { return x }
	x = 20
	return f()
}
