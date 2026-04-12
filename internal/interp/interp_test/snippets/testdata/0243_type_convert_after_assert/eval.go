package main

func identity(v interface{}) interface{} { return v }

func run() int {
	x := identity(uint(42))
	return int(x.(uint))
}
