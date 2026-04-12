package main

type square struct {
	side float64
}

func (s square) area() float64 {
	return s.side * s.side
}

func (s square) name() string {
	return "square"
}
