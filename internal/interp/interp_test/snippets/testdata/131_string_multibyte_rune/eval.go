package main

func run() int {
	s := "héllo"
	r := []rune(s)
	return len(r)*10 + len(s)
}
