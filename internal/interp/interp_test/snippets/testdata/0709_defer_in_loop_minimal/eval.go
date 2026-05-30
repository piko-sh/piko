package main

import "fmt"

func deferLoop() string {
	var out string
	collector := func(value int) {
		out += fmt.Sprintf("%d,", value)
	}
	for i := 0; i < 3; i++ {
		defer collector(i)
	}
	return ""
}

func run() string {
	_ = deferLoop()
	return "ran"
}
