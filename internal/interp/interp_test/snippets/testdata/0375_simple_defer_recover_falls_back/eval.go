package main

func cleanup(s []int) {
	s[0] = 99
}

func run() int {
	s := []int{0}
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	defer cleanup(s)
	s[0] = 1
	return s[0]
}
