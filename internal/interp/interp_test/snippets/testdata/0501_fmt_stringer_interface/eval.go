package main

import "fmt"

type Colour int

const (
	Red Colour = iota
	Green
	Blue
)

func (c Colour) String() string {
	switch c {
	case Red:
		return "red"
	case Green:
		return "green"
	case Blue:
		return "blue"
	}
	return "unknown"
}

func run() string {
	return fmt.Sprintf("%s %s %s", Red, Green, Blue)
}
