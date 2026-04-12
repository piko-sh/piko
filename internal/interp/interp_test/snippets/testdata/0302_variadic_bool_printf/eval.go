package main

import "strconv"

func formatAll(parts ...any) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += " "
		}
		switch v := part.(type) {
		case bool:
			out += strconv.FormatBool(v)
		case string:
			out += v
		}
	}
	return out
}

func run() string {
	s := "hello"
	return formatAll(s != "", s == "")
}
