package main

import (
	"sort"
	"strings"
)

func run() string {
	s := []string{"banana", "apple", "cherry"}
	sort.Strings(s)
	return strings.Join(s, ",")
}
