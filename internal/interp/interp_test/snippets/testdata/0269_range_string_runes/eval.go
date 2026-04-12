package main

func run() int {
	sum := 0
	for _, r := range "ABC" {
		sum += int(r)
	}
	return sum
}
