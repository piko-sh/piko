package main

func compute() int {
	fn := func() int { return 5 }
	defer func() {
		fn = func() int { return 10 }
	}()
	return fn()
}

func run() int {
	return compute()
}
