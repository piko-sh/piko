package main

const (
	flagA = 1 << iota
	flagB
	flagC
	flagD
)

func run() int {
	mask := flagB | flagD
	return mask
}
