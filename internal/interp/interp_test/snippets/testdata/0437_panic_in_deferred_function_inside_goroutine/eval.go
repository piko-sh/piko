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
		defer func() {
			panic("inner")
		}()
		panic("outer")
	}()
	msg := <-ch
	if msg == "inner" {
		return 1
	}
	return 0
}
