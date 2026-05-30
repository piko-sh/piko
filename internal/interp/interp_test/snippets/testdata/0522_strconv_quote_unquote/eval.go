package main

import "strconv"

func run() string {
	q := strconv.Quote(`hello "world"`)
	u, err := strconv.Unquote(q)
	if err != nil {
		return "err"
	}
	return u
}
