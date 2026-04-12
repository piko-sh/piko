package main

func run() int {
	a := make([]float64, 16)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
	}
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i]
	}
	return int(sum)
}
