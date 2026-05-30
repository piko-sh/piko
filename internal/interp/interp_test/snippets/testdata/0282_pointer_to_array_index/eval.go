package main

func run() int {
	a := [4]int{10, 20, 30, 40}
	p := &a
	p[3] = 99
	total := 0
	for i := 0; i < len(*p); i++ {
		total += p[i]
	}
	m := map[string]*[3]int{"row": {1, 2, 3}}
	m["row"][0] = 7
	total += m["row"][0] + m["row"][1] + m["row"][2]
	return total
}
