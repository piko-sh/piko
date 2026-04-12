package main

func run() int {
	var s string
	done := make(chan struct{})
	go func() {
		s = "hello-" + "world"
		close(done)
	}()
	<-done
	if s == "hello-world" {
		return 1
	}
	return 0
}
