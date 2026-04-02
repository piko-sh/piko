package main

import "slices"

func run() bool {
	return slices.Contains([]string{"alpha", "beta", "gamma"}, "beta")
}
