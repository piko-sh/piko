package main

type Box struct {
	N int
}

func run() int {
	b := Box{N: 7}
	f := func() int { return b.N }
	b = Box{N: 99}
	return f()
}
