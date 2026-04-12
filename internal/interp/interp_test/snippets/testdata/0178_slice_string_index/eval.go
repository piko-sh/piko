package main

func run() int {
	s := []string{"hello", "world", "foo"}
	s[1] = "bar"
	r := 0
	if s[0] == "hello" {
		r += 1
	}
	if s[1] == "bar" {
		r += 2
	}
	if s[2] == "foo" {
		r += 4
	}
	return r
}
