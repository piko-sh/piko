package main

func run() int {
	ch := make(chan int, 5)
	for i := range 5 {
		go func() {
			ch <- i
		}()
	}
	sum := 0
	for range 5 {
		sum += <-ch
	}
	return sum
}
