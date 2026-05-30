package main

func outer() func() int {
	x := 0
	inc := func() {
		x++
	}
	get := func() int {
		return x
	}
	inc()
	inc()
	inc()
	return get
}

func run() int {
	return outer()()
}
