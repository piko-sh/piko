package main

import "strings"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	_ = strings.Map(func(r rune) rune {
		panic("map boom")
	}, "abc")
	return 99
}
