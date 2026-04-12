package main

func append3(received []byte) []byte {
	received = append(received, 1)
	received = append(received, 2)
	received = append(received, 3)
	return received
}

func run() string {
	original := make([]byte, 0, 8)
	original = append(original, 99)
	extended := append3(original)
	if len(original) != 1 {
		return "original length changed"
	}
	if original[0] != 99 {
		return "original[0] changed"
	}
	if len(extended) != 4 {
		return "extended length wrong"
	}
	if extended[0] != 99 || extended[1] != 1 || extended[2] != 2 || extended[3] != 3 {
		return "extended contents wrong"
	}
	return "ok"
}
