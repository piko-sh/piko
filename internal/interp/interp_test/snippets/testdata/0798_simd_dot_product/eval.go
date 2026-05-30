package main

func run() int {
	a := make([]float64, 10)
	b := make([]float64, 10)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
		b[i] = float64(10 - i)
	}
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return int(sum * 100)
}
