package main

func run() int {
	srcInts := []int{1, 2, 3, 4, 5}
	dstInts := make([]int, 5)
	nInts := copy(dstInts, srcInts)

	srcFloats := []float64{1.5, 2.5, 3.5}
	dstFloats := make([]float64, 3)
	nFloats := copy(dstFloats, srcFloats)

	srcStrings := []string{"a", "b"}
	dstStrings := make([]string, 2)
	nStrings := copy(dstStrings, srcStrings)

	srcBytes := []byte{10, 20, 30, 40}
	dstBytes := make([]byte, 4)
	nBytes := copy(dstBytes, srcBytes)

	total := 0
	for _, value := range dstInts {
		total += value
	}
	for _, value := range dstFloats {
		total += int(value)
	}
	for _, value := range dstStrings {
		total += len(value)
	}
	for _, value := range dstBytes {
		total += int(value)
	}
	total += nInts + nFloats + nStrings + nBytes
	return total
}
