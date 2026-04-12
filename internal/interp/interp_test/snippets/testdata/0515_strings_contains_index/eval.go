package main

import (
	"strconv"
	"strings"
)

func run() string {
	c := strings.Contains("hello world", "world")
	i := strings.Index("hello world", "world")
	return strconv.FormatBool(c) + ":" + strconv.Itoa(i)
}
