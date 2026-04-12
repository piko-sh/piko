package main

type Point struct {
	X int
	Y int
}

func run() int {
	p := &Point{X: 5, Y: 10}
	return p.X + p.Y
}
