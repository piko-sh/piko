package main

const (
	kb = 1024
	mb = kb * kb
	gb = mb * kb
)

func run() int {
	return gb / mb
}
