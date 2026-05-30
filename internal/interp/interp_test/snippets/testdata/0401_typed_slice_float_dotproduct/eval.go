package main

func run() float64 {
	a := make([]float64, 4)
	b := make([]float64, 4)
	a[0] = 1.0
	a[1] = 2.0
	a[2] = 3.0
	a[3] = 4.0
	b[0] = 0.5
	b[1] = 1.5
	b[2] = 2.5
	b[3] = 3.5
	var sum float64
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}
