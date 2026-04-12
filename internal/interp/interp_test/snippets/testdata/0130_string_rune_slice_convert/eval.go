package main

func run() string {
	s := "Hello"
	r := []rune(s)
	r[0] = 'J'
	return string(r)
}
