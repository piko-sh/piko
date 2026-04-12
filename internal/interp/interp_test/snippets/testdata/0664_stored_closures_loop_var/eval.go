package main

func run() string {
	closures := []func() int{}

	for i := 0; i < 4; i++ {
		closures = append(closures, func() int { return i })
	}

	for i := range 4 {
		closures = append(closures, func() int { return i * 10 })
	}

	result := ""
	for _, fn := range closures {
		result += intToStr(fn())
		result += ","
	}
	return result
}

func intToStr(value int) string {
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
