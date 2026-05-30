package main

func evenInts(limit int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		for i := 0; i < limit; i += 2 {
			if !yield(i) {
				return
			}
		}
	}
}

func pairs(items []string) func(yield func(int, string) bool) {
	return func(yield func(int, string) bool) {
		for i, v := range items {
			if !yield(i, v) {
				return
			}
		}
	}
}
