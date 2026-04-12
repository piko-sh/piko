package main

type Point struct {
	X, Y int
}

func run() int {
	m := map[string]Point{
		"a": {X: 1, Y: 2},
		"b": {X: 3, Y: 4},
	}
	a := m["a"]
	b := m["b"]
	return a.X + a.Y + b.X + b.Y
}
