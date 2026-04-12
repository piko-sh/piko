package main

func outer() func() func() int {
	x := 10
	return func() func() int {
		y := 20
		return func() int {
			return x + y
		}
	}
}

func run() int {
	f := outer()
	g := f()
	return g()
}
