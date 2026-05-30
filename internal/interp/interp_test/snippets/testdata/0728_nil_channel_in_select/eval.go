package main

import "fmt"

func run() string {
	result := ""
	var nilCh chan int
	bufCh := make(chan int, 1)
	bufCh <- 7

	select {
	case v := <-bufCh:
		result += fmt.Sprintf("got=%d;", v)
	case v := <-nilCh:
		result += fmt.Sprintf("nil_path=%d;", v)
	}

	select {
	case nilCh <- 1:
		result += "sent_nil;"
	default:
		result += "default_send;"
	}

	select {
	case v := <-nilCh:
		result += fmt.Sprintf("nil_recv=%d;", v)
	default:
		result += "default_recv"
	}

	return result
}
