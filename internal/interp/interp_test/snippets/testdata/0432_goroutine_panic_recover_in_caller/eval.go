package main

func run() int {
	ch := make(chan int, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- 7
				return
			}
			ch <- 0
		}()
		panic("boom")
	}()
	return <-ch
}
