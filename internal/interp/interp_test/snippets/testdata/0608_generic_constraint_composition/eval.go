package main

type Numeric interface {
	~int | ~int64 | ~float64
}

func sum[T Numeric](xs []T) T {
	var total T
	for _, x := range xs {
		total += x
	}
	return total
}

func run() int {
	return int(sum([]int{1, 2, 3, 4, 5}))
}
