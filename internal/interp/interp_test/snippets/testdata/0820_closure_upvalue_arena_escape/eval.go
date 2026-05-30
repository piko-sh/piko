package main

func makeReader(value int) func() string {
	digits := [4]byte{}
	digits[0] = byte('a' + value)
	digits[1] = byte('a' + value + 1)
	digits[2] = byte('a' + value + 2)
	digits[3] = byte('a' + value + 3)
	label := string(digits[:])
	return func() string {
		return label
	}
}

func churnArena() int {
	scratch := 0
	for index := 0; index < 64; index++ {
		bytes := [8]byte{}
		bytes[0] = byte(index)
		bytes[1] = byte(index + 1)
		filler := string(bytes[:])
		for byteIndex := 0; byteIndex < len(filler); byteIndex++ {
			scratch = scratch*7 + int(filler[byteIndex])
		}
	}
	return scratch
}

func run() int {
	reader := makeReader(3)
	churn := churnArena()
	label := reader()
	checksum := churn
	for byteIndex := 0; byteIndex < len(label); byteIndex++ {
		checksum = checksum*31 + int(label[byteIndex])
	}
	return checksum
}
