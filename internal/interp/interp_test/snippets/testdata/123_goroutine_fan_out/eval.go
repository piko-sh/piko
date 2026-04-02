package main

func run() int {
	ch := make(chan int, 3)
	for i := 0; i < 3; i++ {
		go func(v int) {
			ch <- v * 10
		}(i)
	}
	return <-ch + <-ch + <-ch
}
