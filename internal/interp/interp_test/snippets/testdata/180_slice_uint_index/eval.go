package main

func run() int {
	s := []uint{10, 20, 30}
	s[0] = 5
	var total uint
	for _, v := range s {
		total += v
	}
	return int(total)
}
