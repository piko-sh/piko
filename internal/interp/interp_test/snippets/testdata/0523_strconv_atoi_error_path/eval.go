package main

import "strconv"

func run() int {
	_, err := strconv.Atoi("not-a-number")
	if err != nil {
		return 1
	}
	return 0
}
