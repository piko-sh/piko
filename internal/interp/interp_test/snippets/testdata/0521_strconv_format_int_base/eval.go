package main

import "strconv"

func run() string {
	a := strconv.FormatInt(255, 16)
	b := strconv.FormatInt(8, 2)
	c := strconv.FormatInt(-42, 10)
	return a + "|" + b + "|" + c
}
