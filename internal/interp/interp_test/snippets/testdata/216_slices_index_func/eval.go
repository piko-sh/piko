package main

import "slices"

func run() int {
	s := []string{"alpha", "beta", "gamma"}
	return slices.IndexFunc(s, func(v string) bool { return v == "beta" })
}
