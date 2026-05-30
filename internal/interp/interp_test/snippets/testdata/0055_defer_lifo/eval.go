package main

func deferTest(s []int) {
	defer func() {
		s[0] = s[0] * 10
	}()
	defer func() {
		s[0] = s[0] + 5
	}()
	s[0] = 1
}

func run() int {
	s := make([]int, 1)
	deferTest(s)
	return s[0]
}
