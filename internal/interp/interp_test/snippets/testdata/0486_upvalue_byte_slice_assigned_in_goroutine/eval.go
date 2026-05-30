package main

func run() int {
	var b []byte
	done := make(chan struct{})
	go func() {
		b = []byte("hello")
		close(done)
	}()
	<-done
	if len(b) == 5 && b[0] == 'h' {
		return 1
	}
	return 0
}
