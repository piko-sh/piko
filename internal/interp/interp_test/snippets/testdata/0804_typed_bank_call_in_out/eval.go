package main

func scaleAll(values []float64, k float64) {
	for i := 0; i < len(values); i++ {
		values[i] *= k
	}
}

func sumAll(values []float64) float64 {
	total := 0.0
	for i := 0; i < len(values); i++ {
		total += values[i]
	}
	return total
}

func run() int {
	data := make([]float64, 6)
	for i := 0; i < len(data); i++ {
		data[i] = float64(i + 1)
	}
	scaleAll(data, 2.5)
	total := sumAll(data)
	return int(total * 10)
}
