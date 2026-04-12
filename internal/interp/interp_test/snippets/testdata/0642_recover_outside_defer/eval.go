package main

func run() int {
	r := recover()
	if r == nil {
		return 1
	}
	return 0
}
