package main

func run() int {
	r := make([]bool, 10)
	for i := 0; i < 10; i++ {
		r[i] = i%3 == 0
	}
	count := 0
	for _, b := range r {
		if b {
			count++
		}
	}
	return count
}
