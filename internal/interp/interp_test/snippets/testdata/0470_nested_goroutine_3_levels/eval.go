package main

func run() int {
	level1 := make(chan int, 1)
	go func() {
		level2 := make(chan int, 1)
		go func() {
			level3 := make(chan int, 1)
			go func() {
				level3 <- 30
			}()
			level2 <- <-level3 + 2
		}()
		level1 <- <-level2 + 1
	}()
	return <-level1
}
