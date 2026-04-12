package main

import "fmt"

func sendOnClosed() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	c := make(chan int, 1)
	close(c)
	c <- 42
	return ""
}

func recvFromClosedBuffered() string {
	c := make(chan int, 3)
	c <- 1
	c <- 2
	close(c)

	result := ""
	for value := range c {
		result += fmt.Sprintf("%d,", value)
	}
	v, ok := <-c
	result += fmt.Sprintf("post:v=%d/ok=%v", v, ok)
	return result
}

func recvFromClosedUnbuffered() string {
	c := make(chan int)
	close(c)
	v, ok := <-c
	return fmt.Sprintf("v=%d/ok=%v", v, ok)
}

func run() string {
	out := "send:" + sendOnClosed() + ";"
	out += "recvBuf:" + recvFromClosedBuffered() + ";"
	out += "recvUnbuf:" + recvFromClosedUnbuffered()
	return out
}
