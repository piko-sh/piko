package main

func find(target int) int {
	iter := func(yield func(int) bool) {
		for i := 0; i < 10; i++ {
			if !yield(i) {
				return
			}
		}
	}
	for v := range iter {
		if v == target {
			return v * 10
		}
	}
	return -1
}

func run() int {
	return find(3)
}
