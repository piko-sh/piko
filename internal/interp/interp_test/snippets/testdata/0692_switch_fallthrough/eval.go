package main

import "fmt"

func classify(n int) string {
	result := ""
	switch n {
	case 1:
		result += "one;"
		fallthrough
	case 2:
		result += "two;"
		fallthrough
	case 3:
		result += "three;"
	case 4:
		result += "four;"
	case 5:
		result += "five;"
		fallthrough
	case 6:
		result += "six"
	}
	return result
}

func run() string {
	out := ""
	for n := 1; n <= 6; n++ {
		out += fmt.Sprintf("%d:%s|", n, classify(n))
	}
	return out
}
