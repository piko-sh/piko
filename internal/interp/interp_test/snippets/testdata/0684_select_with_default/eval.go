package main

import "fmt"

func tryRecv(ch chan int) (int, bool) {
	select {
	case v := <-ch:
		return v, true
	default:
		return 0, false
	}
}

func trySend(ch chan int, value int) bool {
	select {
	case ch <- value:
		return true
	default:
		return false
	}
}

func selectMultiCase(a, b chan int) string {
	select {
	case v := <-a:
		return fmt.Sprintf("a=%d", v)
	case v := <-b:
		return fmt.Sprintf("b=%d", v)
	default:
		return "none"
	}
}

func run() string {
	result := ""

	empty := make(chan int)
	v, ok := tryRecv(empty)
	result += fmt.Sprintf("empty:v=%d/ok=%v;", v, ok)

	buffered := make(chan int, 1)
	buffered <- 42
	v, ok = tryRecv(buffered)
	result += fmt.Sprintf("buf1:v=%d/ok=%v;", v, ok)
	v, ok = tryRecv(buffered)
	result += fmt.Sprintf("buf2:v=%d/ok=%v;", v, ok)

	full := make(chan int, 1)
	full <- 1
	sent := trySend(full, 2)
	result += fmt.Sprintf("trySendFull:%v;", sent)

	a := make(chan int, 1)
	b := make(chan int, 1)
	result += "twoEmpty:" + selectMultiCase(a, b) + ";"

	a <- 5
	result += "aReady:" + selectMultiCase(a, b)

	return result
}
