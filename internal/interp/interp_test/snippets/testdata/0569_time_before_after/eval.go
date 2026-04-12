package main

import "time"

func run() int {
	a := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if a.Before(b) && b.After(a) {
		return 1
	}
	return 0
}
