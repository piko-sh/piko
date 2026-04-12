package main

func run() int {
	var f func() int
	for i := range []int{10, 20, 30} {
		i := i
		f = func() int { return i }
	}
	return f()
}
