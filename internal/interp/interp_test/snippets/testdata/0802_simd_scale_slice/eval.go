package main

func run() int {
	a := make([]float64, 13)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
	}
	k := 2.5
	for i := 0; i < len(a); i++ {
		a[i] *= k
	}
	total := 0.0
	for i := 0; i < len(a); i++ {
		total += a[i]
	}
	return int(total * 10)
}
