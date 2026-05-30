package main

func run() int {
	flags := make([]bool, 8)
	flags[0] = true
	flags[3] = true
	flags[5] = true
	flags[7] = true
	count := 0
	for i := 0; i < len(flags); i++ {
		if flags[i] {
			count++
		}
	}
	return count
}
