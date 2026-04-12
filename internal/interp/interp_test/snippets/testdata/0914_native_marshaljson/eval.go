package main

import (
	"encoding/json"
	"fmt"
)

type Point struct {
	X, Y int
}

func (p Point) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"x":%d,"y":%d,"sum":%d}`, p.X, p.Y, p.X+p.Y)), nil
}

func run() string {
	p := Point{X: 3, Y: 4}
	out, err := json.Marshal(p)
	if err != nil {
		return "ERR " + err.Error()
	}
	return string(out)
}
