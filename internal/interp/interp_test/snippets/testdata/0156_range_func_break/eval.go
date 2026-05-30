package main

func run() int {
	iter := func(yield func(int) bool) {
		for i := 0; i < 10; i++ {
			if !yield(i) {
				return
			}
		}
	}
	sum := 0
	for v := range iter {
		if v >= 3 {
			break
		}
		sum += v
	}
	return sum
}
