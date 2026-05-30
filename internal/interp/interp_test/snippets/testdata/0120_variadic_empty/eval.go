package main

func count(arguments ...int) int {
	return len(arguments)
}

func run() int {
	return count()
}
