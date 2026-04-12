package main

func setIt(p *int) {
	*p = 42
}

func compute() (n int) {
	setIt(&n)
	return
}

func run() int {
	return compute()
}
