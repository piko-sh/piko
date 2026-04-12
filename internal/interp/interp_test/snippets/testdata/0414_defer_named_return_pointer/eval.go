package main

func adjust(p *int) {
	*p += 100
}

func compute() (result int) {
	defer func() {
		adjust(&result)
	}()
	result = 10
	return
}

func run() int {
	return compute()
}
