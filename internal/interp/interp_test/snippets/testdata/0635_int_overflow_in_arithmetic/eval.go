package main

func run() int {
	var a int64 = 9223372036854775807
	b := a + 1
	if b < 0 {
		return 1
	}
	return 0
}
