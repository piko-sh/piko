package main

func intIntLookup(m map[int]int, k int) int {
	v, ok := m[k]
	if !ok {
		return -1
	}
	return v
}

func stringStringLookup(m map[string]string, k string) string {
	v, ok := m[k]
	if !ok {
		return "missing"
	}
	return v
}

func stringIntLookup(m map[string]int, k string) int {
	v, ok := m[k]
	if !ok {
		return -1
	}
	return v
}

func intStringLookup(m map[int]string, k int) string {
	v, ok := m[k]
	if !ok {
		return "missing"
	}
	return v
}

func run() string {
	resultA := intIntLookup(map[int]int{1: 10, 2: 20}, 2)
	resultB := intIntLookup(map[int]int{1: 10, 2: 20}, 3)
	resultC := stringStringLookup(map[string]string{"a": "alpha"}, "a")
	resultD := stringStringLookup(map[string]string{"a": "alpha"}, "z")
	resultE := stringIntLookup(map[string]int{"k": 42}, "k")
	resultF := intStringLookup(map[int]string{7: "seven"}, 99)

	out := ""
	out += stringFromInt(resultA)
	out += "/"
	out += stringFromInt(resultB)
	out += "/"
	out += resultC
	out += "/"
	out += resultD
	out += "/"
	out += stringFromInt(resultE)
	out += "/"
	out += resultF
	return out
}

func stringFromInt(value int) string {
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
