package main

import "strconv"

func run() int {
	v, err := strconv.ParseInt("42", 10, 64)
	if err != nil {
		return -1
	}
	return int(v)
}
