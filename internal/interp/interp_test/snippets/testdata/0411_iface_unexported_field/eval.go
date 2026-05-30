package main

type shape interface {
	area() float64
}

type circle struct {
	radius float64
}

func (c circle) area() float64 {
	return 3.1416 * c.radius * c.radius
}

func run() float64 {
	var s shape = circle{radius: 5}
	return s.area()
}
