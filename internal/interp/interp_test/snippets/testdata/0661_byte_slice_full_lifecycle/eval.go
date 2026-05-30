package main

func run() string {
	buffer := make([]byte, 10)

	for index := 0; index < len(buffer); index++ {
		buffer[index] = byte('a' + index)
	}

	prefix := buffer[0:5]
	suffix := buffer[5:10]

	var combined []byte
	for _, b := range prefix {
		combined = append(combined, b)
	}
	for _, b := range suffix {
		combined = append(combined, b)
	}

	combined[3] = 'X'

	return string(combined)
}
