package main

import "time"

func run() int {
	d := 5*time.Second + 200*time.Millisecond
	if d == 5200*time.Millisecond {
		return 1
	}
	return 0
}
