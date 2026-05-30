package main

func f() int {
	x := 0
again:
	x++
	if x < 5 {
		goto again
	}
	return x
}

func run() int {
	return f()
}
