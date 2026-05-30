package main

func runPipeline(n int) int {
	in := make(chan int, n)
	out := make(chan int, n)
	go func() {
		for i := 1; i <= n; i++ {
			in <- i
		}
		close(in)
	}()
	go square(in, out)
	sum := 0
	for v := range out {
		sum += v
	}
	return sum
}
