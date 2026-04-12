package main

import "fmt"

type FailureReason struct {
	Code    int
	Message string
}

func raise() {
	panic(FailureReason{Code: 42, Message: "bad request"})
}

func raiseInt() {
	panic(99)
}

func raiseBool() {
	panic(true)
}

func raiseSlice() {
	panic([]int{1, 2, 3})
}

func recoverWith(action func()) any {
	defer func() {
		recover()
	}()
	var captured any
	defer func() {
		captured = recover()
	}()
	action()
	return captured
}

func recoverViaInner(action func()) any {
	var captured any
	func() {
		defer func() {
			captured = recover()
		}()
		action()
	}()
	return captured
}

func run() string {
	a := recoverViaInner(raise)
	b := recoverViaInner(raiseInt)
	c := recoverViaInner(raiseBool)
	d := recoverViaInner(raiseSlice)

	return fmt.Sprintf("a=%v;b=%v;c=%v;d=%v", a, b, c, d)
}
