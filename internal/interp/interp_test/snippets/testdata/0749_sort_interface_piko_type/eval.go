package main

import (
	"fmt"
	"sort"
)

type ByLen []string

func (b ByLen) Len() int           { return len(b) }
func (b ByLen) Less(i, j int) bool { return len(b[i]) < len(b[j]) }
func (b ByLen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func run() string {
	items := ByLen{"hello", "hi", "greetings", "yo", "good day"}
	var iface sort.Interface = items
	sort.Sort(iface)
	result := fmt.Sprintf("count=%d;", len(items))
	for _, s := range items {
		result += s + ";"
	}
	return result
}
