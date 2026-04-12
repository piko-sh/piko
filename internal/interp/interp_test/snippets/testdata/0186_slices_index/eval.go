package main

import "slices"

func run() int {
	return slices.Index([]string{"alpha", "beta", "gamma"}, "gamma")
}
