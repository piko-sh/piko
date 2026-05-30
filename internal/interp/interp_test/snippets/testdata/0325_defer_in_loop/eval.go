package main

func collect() []int {
	out := make([]int, 0, 8)
	defer func() { out = append(out, 999) }()
	for i := 1; i <= 3; i++ {
		out = append(out, i*10)
	}
	return out
}

func run() int {
	out := collect()
	total := 0
	for _, v := range out {
		total += v
	}
	return total
}
