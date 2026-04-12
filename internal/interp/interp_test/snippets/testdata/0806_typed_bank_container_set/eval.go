package main

func run() int {
	ints := []int{10, 20, 30}
	floats := []float64{1.5, 2.5}
	strings := []string{"a", "b", "c"}

	mapOut := map[string]any{}
	mapOut["i"] = ints
	mapOut["f"] = floats
	mapOut["s"] = strings

	nested := make([][]int, 2)
	nested[0] = ints
	nested[1] = []int{40, 50}

	total := 0
	for _, value := range nested[0] {
		total += value
	}
	for _, value := range nested[1] {
		total += value
	}
	total += len(mapOut)
	total += len(floats)
	total += len(strings)
	return total
}
