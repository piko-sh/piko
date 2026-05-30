package main

func run() int {
	done := make(chan struct{})
	result := 0
	go func() {
		result = 42
		close(done)
	}()
	<-done
	return result
}
