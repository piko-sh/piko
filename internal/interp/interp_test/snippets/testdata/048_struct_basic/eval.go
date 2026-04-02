package main

type Point struct {
	X int
	Y int
}

func run() int {
	p := Point{X: 3, Y: 4}
	return p.X + p.Y
}
