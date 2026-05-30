package main

func minmax(a int, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}

func run() int {
	lo, hi := minmax(42, 7)
	return hi - lo
}
