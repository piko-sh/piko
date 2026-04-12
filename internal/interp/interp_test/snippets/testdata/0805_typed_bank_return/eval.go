package main

func makeFloats(n int) []float64 {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = float64(i+1) * 0.5
	}
	return out
}

func run() int {
	data := makeFloats(8)
	total := 0.0
	for i := 0; i < len(data); i++ {
		total += data[i]
	}
	return int(total * 100)
}
