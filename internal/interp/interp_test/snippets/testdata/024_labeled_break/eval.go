package main

func count() int {
	sum := 0
outer:
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if j == 2 {
				break outer
			}
			sum++
		}
	}
	return sum
}

func run() int {
	return count()
}
