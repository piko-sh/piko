package main

import "regexp"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	re := regexp.MustCompile(`\d+`)
	_ = re.ReplaceAllStringFunc("a 1 b 2 c", func(m string) string {
		panic("replace boom")
	})
	return 99
}
