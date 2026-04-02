package main

func run() int {
	iter := func(yield func(int) bool) {
		for i := 0; i < 5; i++ {
			if !yield(i) {
				return
			}
		}
	}
	sum := 0
	for v := range iter {
		if v%2 == 0 {
			continue
		}
		sum += v
	}
	return sum
}
