package main

func deferTest(s []int) {
	defer func() {
		s[0] = 42
	}()
	s[0] = 1
}

func run() int {
	s := make([]int, 1)
	deferTest(s)
	return s[0]
}
