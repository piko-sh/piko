package main

func run() int {
	bits := make([]bool, 16)
	for i := 0; i < len(bits); i++ {
		if i%3 == 0 {
			bits[i] = true
		}
	}
	count := 0
	for _, v := range bits {
		if v {
			count++
		}
	}
	return count
}
