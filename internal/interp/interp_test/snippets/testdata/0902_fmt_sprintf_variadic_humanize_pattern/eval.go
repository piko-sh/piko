package main

import "fmt"

type magnitude struct {
	threshold int64
	format    string
	divBy     int64
}

var table = []magnitude{
	{60, "%d seconds %s", 1},
	{3600, "%d minutes %s", 60},
}

func render(diff int64, label string) string {
	var mag magnitude
	for _, m := range table {
		if diff < m.threshold {
			mag = m
			break
		}
	}
	args := []interface{}{}
	escaped := false
	for _, ch := range mag.format {
		if escaped {
			switch ch {
			case 's':
				args = append(args, label)
			case 'd':
				args = append(args, diff/mag.divBy)
			}
			escaped = false
		} else {
			escaped = ch == '%'
		}
	}
	return fmt.Sprintf(mag.format, args...)
}

func run() string {
	return render(30, "ago") + "|" + render(120, "from now")
}
