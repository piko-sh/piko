package main

func run() int {
	ch1 := make(chan int, 20)
	ch2 := make(chan int, 20)
	for i := 0; i < 20; i++ {
		ch1 <- 1
		ch2 <- 1
	}
	count1, count2 := 0, 0
	for i := 0; i < 20; i++ {
		select {
		case <-ch1:
			count1++
		case <-ch2:
			count2++
		}
	}
	if count1 > 0 && count2 > 0 {
		return 1
	}
	return 0
}
