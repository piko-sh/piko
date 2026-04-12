package main

type Point struct {
	X int
	Y int
}

func run() int {
	p := Point{3, 7}
	return p.X*10 + p.Y
}
