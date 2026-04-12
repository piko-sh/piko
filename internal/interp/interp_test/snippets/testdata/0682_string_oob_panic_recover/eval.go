package main

import "fmt"

func panicOnStringIndex() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	s := "abc"
	_ = s[10]
	return ""
}

func panicOnSliceIndex() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	s := []int{1, 2, 3}
	_ = s[10]
	return ""
}

func panicOnSliceSlice() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	s := []int{1, 2, 3}
	_ = s[2:10]
	return ""
}

func panicOnSliceSliceInverted() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	s := []int{1, 2, 3}
	low := 2
	high := 1
	_ = s[low:high]
	return ""
}

func panicOnNilMapWrite() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	var m map[string]int
	m["k"] = 1
	return ""
}

func panicOnDivByZero() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	x := 10
	y := 0
	_ = x / y
	return ""
}

func panicOnNilPtrDeref() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	var p *int
	_ = *p
	return ""
}

func run() string {
	return "stridx:" + panicOnStringIndex() +
		";sliceidx:" + panicOnSliceIndex() +
		";slicelice:" + panicOnSliceSlice() +
		";sliceinverted:" + panicOnSliceSliceInverted() +
		";nilmap:" + panicOnNilMapWrite() +
		";divzero:" + panicOnDivByZero() +
		";nilptr:" + panicOnNilPtrDeref()
}
