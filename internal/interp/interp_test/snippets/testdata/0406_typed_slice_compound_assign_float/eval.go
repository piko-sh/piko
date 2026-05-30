package main

func run() float64 {
	values := make([]float64, 5)
	values[0] = 10.0
	values[1] = 20.0
	values[2] = 30.0
	values[3] = 40.0
	values[4] = 50.0
	for i := 0; i < len(values); i++ {
		values[i] *= 2.0
	}
	var total float64
	for _, v := range values {
		total += v
	}
	return total
}
