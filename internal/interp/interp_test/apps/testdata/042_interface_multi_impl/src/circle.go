package main

type circle struct {
	radius float64
}

func (c circle) Area() float64 {
	return 3.1416 * c.radius * c.radius
}

func (c circle) Name() string {
	return "circle"
}
