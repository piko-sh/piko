package main

func run() int {
	s := "Hi!"
	total := 0
	for _, ch := range s {
		total += int(ch)
	}
	return total
}
