package main

func run() int {
	a := [5]int{10, 20, 30, 40, 50}
	p := &a

	total := 0
	for i, v := range p {
		total += v * (i + 1)
	}

	total += len(p)
	total += cap(p)

	mid := p[1:4]
	for _, v := range mid {
		total += v
	}

	return total
}
