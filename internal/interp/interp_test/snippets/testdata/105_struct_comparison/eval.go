package main

type Point struct {
	X int
	Y int
}

func run() int {
	p1 := Point{X: 1, Y: 2}
	p2 := Point{X: 1, Y: 2}
	p3 := Point{X: 3, Y: 4}
	r := 0
	if p1 == p2 {
		r += 1
	}
	if p1 != p3 {
		r += 2
	}
	return r
}
