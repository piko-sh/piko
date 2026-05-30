package main

type circle struct {
	radius float64
}

func (c circle) area() float64 {
	return 3.1416 * c.radius * c.radius
}

func (c circle) name() string {
	return "circle"
}
