package main

func run() int {
	x := 0
	f := func() func() func() int {
		return func() func() int {
			return func() int {
				x++
				return x
			}
		}
	}
	g := f()()
	g()
	g()
	return g()
}
