package main

func run() string {
	output := make([]byte, 0, 1024)
	for index := 0; index < 256; index++ {
		output = append(output, byte(index))
	}
	if len(output) != 256 {
		return "wrong length"
	}
	for index := 0; index < 256; index++ {
		if output[index] != byte(index) {
			return "wrong byte"
		}
	}
	return string(output[0:5])
}
