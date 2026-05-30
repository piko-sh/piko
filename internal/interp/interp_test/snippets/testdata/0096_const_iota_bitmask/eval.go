package main

const (
	Read = 1 << iota
	Write
	Execute
)

func run() int {
	return Read | Write | Execute
}
