package main

import "fmt"

func tryDoubleClose() string {
	defer func() {}()
	var caught any
	func() {
		defer func() {
			caught = recover()
		}()
		ch := make(chan int)
		close(ch)
		close(ch)
	}()
	return fmt.Sprintf("%v", caught)
}

func tryCloseNil() string {
	var caught any
	func() {
		defer func() {
			caught = recover()
		}()
		var ch chan int
		close(ch)
	}()
	return fmt.Sprintf("%v", caught)
}

func recvFromNilSelect() string {
	var ch chan int
	select {
	case v := <-ch:
		return fmt.Sprintf("got:%d", v)
	default:
		return "default"
	}
}

func run() string {
	return "double:" + tryDoubleClose() +
		";nil:" + tryCloseNil() +
		";nilRecv:" + recvFromNilSelect()
}
