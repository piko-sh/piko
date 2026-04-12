package main

func run() int {
	a := make(chan int, 3)
	b := make(chan int, 3)
	go func() { a <- 1; a <- 2; close(a) }()
	go func() { b <- 10; b <- 20; close(b) }()
	sum := 0
	openA, openB := true, true
	for openA || openB {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil
				openA = false
				continue
			}
			sum += v
		case v, ok := <-b:
			if !ok {
				b = nil
				openB = false
				continue
			}
			sum += v
		}
	}
	return sum
}
