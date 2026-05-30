package main

func compute() int {
	x := 5
	y := 50
	var p *int
	p = &x
	defer func() {
		p = &y
	}()
	return *p
}

func run() int {
	return compute()
}
