package main

func run() int {
	ch := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if s, ok := r.(string); ok {
					ch <- s
					return
				}
			}
			ch <- ""
		}()
		message := "hello-" + "world"
		panic(message)
	}()
	got := <-ch
	if got == "hello-world" {
		return 1
	}
	return 0
}
