package main

type Rect struct {
	W int
	H int
}

func rectArea(r Rect) int {
	return r.W * r.H
}

type Triangle struct {
	Base   int
	Height int
}

func triangleArea(t Triangle) int {
	return t.Base * t.Height / 2
}
