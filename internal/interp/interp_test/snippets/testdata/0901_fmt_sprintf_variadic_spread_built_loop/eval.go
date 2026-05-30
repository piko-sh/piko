package main

import "fmt"

func run() string {
	format := "%d seconds %s"
	args := []interface{}{}
	escaped := false
	count := 30
	label := "ago"
	for _, ch := range format {
		if escaped {
			switch ch {
			case 's':
				args = append(args, label)
			case 'd':
				args = append(args, count)
			}
			escaped = false
		} else {
			escaped = ch == '%'
		}
	}
	return fmt.Sprintf(format, args...)
}
