package main

func run() int {
	iter := func(yield func(string, int) bool) {
		yield("a", 1)
		yield("b", 2)
		yield("c", 3)
	}
	total := 0
	for _, v := range iter {
		total += v
	}
	return total
}
