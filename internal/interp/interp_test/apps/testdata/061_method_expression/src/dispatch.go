package main

func applyRead(read func(Counter) int, c Counter) int {
	return read(c)
}

func applyBump(bump func(*Counter), c *Counter, times int) {
	for range times {
		bump(c)
	}
}
