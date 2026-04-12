package main

func run() int {
	iter := func(yield func(int) bool) {
		for i := 0; i < 3; i++ {
			if !yield(i) {
				return
			}
		}
	}
	sum := 0
	for a := range iter {
		for b := range iter {
			sum += a*10 + b
		}
	}
	return sum
}
