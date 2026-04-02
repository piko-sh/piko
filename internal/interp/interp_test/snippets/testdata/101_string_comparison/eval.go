package main

func run() int {
	r := 0
	if "abc" < "abd" {
		r += 1
	}
	if "abc" <= "abc" {
		r += 2
	}
	if "abd" > "abc" {
		r += 4
	}
	if "abc" >= "abc" {
		r += 8
	}
	if "abc" == "abc" {
		r += 16
	}
	if "abc" != "abd" {
		r += 32
	}
	return r
}
