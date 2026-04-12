package main

type Rect struct {
	W int
	H int
}

func (r Rect) Area() int {
	return r.W * r.H
}

func run() int {
	r := Rect{W: 6, H: 7}
	return r.Area()
}
