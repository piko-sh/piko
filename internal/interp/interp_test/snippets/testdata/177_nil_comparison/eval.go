package main

func run() int {
	r := 0
	var p *int
	if p == nil {
		r += 1
	}
	x := 42
	p = &x
	if p != nil {
		r += 2
	}
	var s []int
	if s == nil {
		r += 4
	}
	s = []int{1}
	if s != nil {
		r += 8
	}
	return r
}
