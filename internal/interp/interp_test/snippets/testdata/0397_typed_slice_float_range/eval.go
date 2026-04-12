package main

func run() float64 {
	s := make([]float64, 5)
	s[0] = 1.5
	s[1] = 2.5
	s[2] = 4.0
	s[3] = 0.5
	s[4] = 1.0
	var total float64
	for _, v := range s {
		total += v
	}
	return total
}
