package main

func run() int {
	ch := make(chan int, 1)
	ch <- 5
	close(ch)
	v1, ok1 := <-ch
	v2, ok2 := <-ch
	result := v1*1000 + v2*100
	if ok1 {
		result += 10
	}
	if !ok2 {
		result += 1
	}
	return result
}
