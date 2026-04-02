package main

func run() int {
	s := "hello"
	count := 0
	for range s {
		count++
	}
	return count
}
