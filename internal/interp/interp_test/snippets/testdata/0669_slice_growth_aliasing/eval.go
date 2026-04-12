package main

func growBeyondCapacity() (originalFirst, newFirst, originalLen, newLen int) {
	original := make([]int, 3, 3)
	for i := range original {
		original[i] = (i + 1) * 10
	}
	grown := append(original, 99)
	grown[0] = 100
	return original[0], grown[0], len(original), len(grown)
}

func inPlaceAppend() (originalFirst, newFirst, originalLen, newLen int) {
	original := make([]int, 3, 6)
	for i := range original {
		original[i] = (i + 1) * 10
	}
	extended := append(original, 99)
	extended[0] = 100
	return original[0], extended[0], len(original), len(extended)
}

func threeIndexForcesGrow() (originalFirst, newFirst int) {
	original := make([]int, 3, 6)
	original[0] = 7
	limited := original[:3:3]
	grown := append(limited, 99)
	grown[0] = 200
	return original[0], grown[0]
}

func appendThroughOffsetView() (a, b int) {
	original := make([]int, 6, 6)
	for i := range original {
		original[i] = i
	}
	view := original[2:4:4]
	grown := append(view, 999)
	return original[4], grown[2]
}

func run() string {
	a1, a2, l1, l2 := growBeyondCapacity()
	b1, b2, l3, l4 := inPlaceAppend()
	c1, c2 := threeIndexForcesGrow()
	d1, d2 := appendThroughOffsetView()

	result := ""
	result += "grow:" + intToStr(int64(a1)) + "/" + intToStr(int64(a2))
	result += "/" + intToStr(int64(l1)) + "/" + intToStr(int64(l2)) + ";"
	result += "inplace:" + intToStr(int64(b1)) + "/" + intToStr(int64(b2))
	result += "/" + intToStr(int64(l3)) + "/" + intToStr(int64(l4)) + ";"
	result += "3idx:" + intToStr(int64(c1)) + "/" + intToStr(int64(c2)) + ";"
	result += "offset:" + intToStr(int64(d1)) + "/" + intToStr(int64(d2))
	return result
}

func intToStr(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
