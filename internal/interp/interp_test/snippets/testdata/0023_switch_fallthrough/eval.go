package main

func run() int {
	x := 1
	r := 0
	switch x {
	case 1:
		r += 1
		fallthrough
	case 2:
		r += 2
		fallthrough
	case 99:
		r += 4
	}
	return r
}
