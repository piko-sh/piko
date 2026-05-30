package main

func augmentError() (result string) {
	defer func() {
		result = "wrapped(" + result + ")"
	}()
	return "ok"
}

func stackOfDefers() (n int) {
	defer func() { n += 1 }()
	defer func() { n *= 2 }()
	defer func() { n = 5 }()
	return 0
}

func recoverThenMutate() (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = "recovered:" + result
		}
	}()
	result = "before-panic"
	panic("boom")
}

func readModifyWrite() (count int) {
	defer func() {
		count = count*10 + 7
	}()
	return 3
}

func run() string {
	out := ""
	out += augmentError() + ";"
	out += intToStr(int64(stackOfDefers())) + ";"
	out += recoverThenMutate() + ";"
	out += intToStr(int64(readModifyWrite()))
	return out
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
