package main

func run() int {
	r := 0
	if "abc" != "abd" {
		r += 1
	}
	if "abd" > "abc" {
		r += 2
	}
	if "abc" >= "abc" {
		r += 4
	}
	if "abd" >= "abc" {
		r += 8
	}
	if !("abc" != "abc") {
		r += 16
	}
	return r
}
