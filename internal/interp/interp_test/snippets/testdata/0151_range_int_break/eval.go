package main

func run() int {
	s := 0
	for i := range 20 {
		if i >= 5 {
			break
		}
		s += i
	}
	return s
}
