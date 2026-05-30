package main

func run() int {
	ch := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		ch <- i
	}
	close(ch)
	sum := 0
loop:
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				break loop
			}
			sum += v
		}
	}
	return sum
}
