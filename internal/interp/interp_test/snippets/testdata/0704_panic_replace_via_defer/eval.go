package main

import "fmt"

func work() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			_ = r
			panic("replaced")
		}
	}()
	panic("original")
}

func run() string {
	return "got:" + work()
}
