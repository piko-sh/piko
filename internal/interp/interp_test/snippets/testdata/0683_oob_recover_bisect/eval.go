package main

import "fmt"

func tryRecover(action func()) string {
	defer func() {}()
	var caught any
	func() {
		defer func() {
			caught = recover()
		}()
		action()
	}()
	if caught == nil {
		return "no-panic"
	}
	return fmt.Sprintf("%v", caught)
}

func run() string {
	stringIndex := tryRecover(func() {
		index := 10
		s := "abc"
		_ = s[index]
	})

	sliceIndex := tryRecover(func() {
		index := 10
		s := []int{1, 2, 3}
		_ = s[index]
	})

	divZero := tryRecover(func() {
		y := 0
		_ = 10 / y
	})

	nilPtr := tryRecover(func() {
		var p *int
		_ = *p
	})

	nilMap := tryRecover(func() {
		var m map[string]int
		m["k"] = 1
	})

	return "str:" + stringIndex + ";slc:" + sliceIndex + ";div:" + divZero + ";ptr:" + nilPtr + ";map:" + nilMap
}
