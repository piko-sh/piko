package main

func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func run() string {
	bytes := []byte{'A', 'a', '5', '!', 'Z'}
	result := ""
	for index := 0; index < len(bytes); index++ {
		if isUpper(bytes[index]) {
			result += "U"
			continue
		}
		if isDigit(bytes[index]) {
			result += "D"
			continue
		}
		result += "."
	}
	return result
}
