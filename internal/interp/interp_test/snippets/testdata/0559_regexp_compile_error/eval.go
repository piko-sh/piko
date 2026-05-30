package main

import "regexp"

func run() int {
	_, err := regexp.Compile(`(\d`)
	if err != nil {
		return 1
	}
	return 0
}
