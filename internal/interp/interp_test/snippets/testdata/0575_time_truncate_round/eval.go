package main

import "time"

func run() int {
	d := 1530 * time.Millisecond
	tr := d.Truncate(time.Second)
	if tr == time.Second {
		return 1
	}
	return 0
}
