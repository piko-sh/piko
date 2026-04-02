package main

func run() int {
	s := 0
	for i := range 10 {
		if i%2 == 0 {
			continue
		}
		s += i
	}
	return s
}
