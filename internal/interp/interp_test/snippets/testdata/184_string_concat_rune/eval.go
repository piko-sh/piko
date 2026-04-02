package main

func run() int {
	s := ""
	for i := 0; i < 26; i++ {
		s += string(rune('a' + i))
	}
	if s != "abcdefghijklmnopqrstuvwxyz" {
		return 0
	}
	return len(s)
}
