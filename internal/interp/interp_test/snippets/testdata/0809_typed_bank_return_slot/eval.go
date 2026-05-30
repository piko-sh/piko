package main

func makeInts() []int {
	out := make([]int, 0, 3)
	out = append(out, 1)
	out = append(out, 2)
	out = append(out, 3)
	return out
}

func makeFloats() []float64 {
	return []float64{1.5, 2.5, 3.5}
}

func toBytes(text string) []byte {
	return []byte(text)
}

func makeStrings() []string {
	return []string{"a", "bb", "ccc"}
}

func run() int {
	ints := makeInts()
	floats := makeFloats()
	bytes := toBytes("hello")
	strings := makeStrings()

	totalInts := 0
	for _, value := range ints {
		totalInts += value
	}

	totalFloats := 0.0
	for _, value := range floats {
		totalFloats += value
	}

	totalBytes := 0
	for _, value := range bytes {
		totalBytes += int(value)
	}

	totalStringLengths := 0
	for _, value := range strings {
		totalStringLengths += len(value)
	}

	return totalInts + int(totalFloats*10) + totalBytes + totalStringLengths
}
