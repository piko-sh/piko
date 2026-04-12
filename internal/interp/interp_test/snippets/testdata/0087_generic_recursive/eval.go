package main

func sum[T ~int](s []T) T {
	if len(s) == 0 {
		return 0
	}
	return s[0] + sum(s[1:])
}

func run() int {
	return sum([]int{1, 2, 3, 4, 5})
}
