package main

func run() int {
	s := "AB"
	total := 0
	for i, ch := range s {
		total += i*100 + int(ch)
	}
	return total
}
