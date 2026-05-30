package main

import "strconv"

func run() int {
	t, _ := strconv.ParseBool("true")
	f, _ := strconv.ParseBool("0")
	if t && !f {
		return 1
	}
	return 0
}
