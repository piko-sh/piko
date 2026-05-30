package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	ch := make(chan string)
	close(ch)
	ch <- "hello"
	return
}
