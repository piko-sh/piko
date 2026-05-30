package main

func run() int {
	ch := make(chan []int, 1)
	s := []int{1, 2, 3}
	ch <- s
	s[0] = 999
	received := <-ch
	return received[0]
}
