package main

func run() int {
	var total uint = 1
	done := make(chan struct{})
	go func() {
		for i := uint(2); i < 6; i++ {
			total *= i
		}
		close(done)
	}()
	<-done
	return int(total)
}
