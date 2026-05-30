package main

func release(s []int) {
	s[0]++
}

func run() int {
	s := []int{0}
	for i := 0; i < 3; i++ {
		defer release(s)
	}
	return s[0]
}
