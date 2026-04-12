package main

func collect() []int {
	out := []int{1, 2, 3}
	defer func() { out = append(out, 99) }()
	out = append(out, 10)
	return out
}

func run() int {
	values := collect()
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}
