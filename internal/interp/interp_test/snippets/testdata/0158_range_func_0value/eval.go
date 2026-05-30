package main

func run() int {
	iter := func(yield func() bool) {
		yield()
		yield()
		yield()
	}
	count := 0
	for range iter {
		count++
	}
	return count
}
