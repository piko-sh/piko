package main

func run() int {
	ints := []int{1, 2, 3, 4, 5}
	floats := []float64{1.5, 2.5, 3.5}

	doubleInts := func() int {
		total := 0
		for _, value := range ints {
			total += value * 2
		}
		return total
	}

	sumFloats := func() int {
		total := 0.0
		for _, value := range floats {
			total += value
		}
		return int(total * 10)
	}

	mutateInts := func() {
		for i := range ints {
			ints[i] += 10
		}
	}

	first := doubleInts()
	mutateInts()
	second := doubleInts()
	floatSum := sumFloats()

	return first + second + floatSum
}
