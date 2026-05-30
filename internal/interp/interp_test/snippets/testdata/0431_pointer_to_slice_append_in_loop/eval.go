package main

func push(s *[]int, x int) {
	*s = append(*s, x)
}

func run() int {
	s := []int{}
	for i := 0; i < 5; i++ {
		push(&s, i)
	}
	return len(s)
}
