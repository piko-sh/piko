package main

func run() int {
	xs := []int{1, 2, 3, 4, 5}
	for i := range xs {
		p := &xs[i]
		*p *= 10
	}
	total := 0
	for _, v := range xs {
		total += v
	}
	return total
}
