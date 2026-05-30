package main

import "fmt"

func chain() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("outer:%v", r)
		}
	}()
	defer func() {
		_ = recover()
	}()
	defer func() {
		panic("inner")
	}()
	return ""
}

func run() string {
	return "got:" + chain()
}
