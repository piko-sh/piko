package main

import "fmt"

func sandwich() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	defer func() {
		panic("from-defer")
	}()
	return ""
}

func run() string {
	return "got:" + sandwich()
}
