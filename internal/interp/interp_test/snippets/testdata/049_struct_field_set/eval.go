package main

type Point struct {
	X int
	Y int
}

func run() int {
	p := Point{X: 1, Y: 2}
	p.X = 10
	return p.X + p.Y
}
