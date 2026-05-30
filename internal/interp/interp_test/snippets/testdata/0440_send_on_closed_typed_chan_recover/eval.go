package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	ch := make(chan int64, 1)
	ch <- 1
	close(ch)
	ch <- 2
	return
}
