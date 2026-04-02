package main

import (
	"slices"
	"strings"
)

func run() bool {
	a := []string{"Hello", "World"}
	b := []string{"hello", "world"}
	return slices.EqualFunc(a, b, func(x, y string) bool {
		return strings.EqualFold(x, y)
	})
}
