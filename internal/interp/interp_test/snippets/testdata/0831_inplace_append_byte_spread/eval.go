package main

func run() string {
	output := make([]byte, 0, 64)
	chunk1 := []byte("hello")
	chunk2 := []byte(" ")
	chunk3 := []byte("world")
	output = append(output, chunk1...)
	output = append(output, chunk2...)
	output = append(output, chunk3...)
	if len(output) != 11 {
		return "wrong length"
	}
	return string(output)
}
