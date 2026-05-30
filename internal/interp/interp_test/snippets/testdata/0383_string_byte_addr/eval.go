package main

func run() int {
	s := "hello"
	bytes := []byte(s)
	bytes[0] = 'H'
	out := string(bytes)
	return len(out) + int(out[0])
}
