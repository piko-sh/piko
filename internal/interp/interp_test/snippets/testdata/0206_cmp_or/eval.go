package main

import "cmp"

func run() int {
	return cmp.Or(0, 0, 42, 7)
}
