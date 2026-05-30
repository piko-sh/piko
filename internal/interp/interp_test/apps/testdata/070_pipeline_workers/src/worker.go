package main

func square(in <-chan int, out chan<- int) {
	for v := range in {
		out <- v * v
	}
	close(out)
}
