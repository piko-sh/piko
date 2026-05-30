package main

func run() string {
	var captured []int

	func() {
		for i := 0; i < 3; i++ {
			defer func() {
				captured = append(captured, i)
			}()
		}
	}()

	result := ""
	for _, value := range captured {
		result += intToStr(value)
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
