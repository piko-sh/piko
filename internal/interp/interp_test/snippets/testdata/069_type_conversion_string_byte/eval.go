package main

func run() string {
	s := "hello"
	b := []byte(s)
	b[0] = 'H'
	return string(b)
}
