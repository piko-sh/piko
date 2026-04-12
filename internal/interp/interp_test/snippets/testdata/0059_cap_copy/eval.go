package main

func run() int {
	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, 3)
	n := copy(dst, src)
	return dst[0] + dst[1] + dst[2] + n
}
