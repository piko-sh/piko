package main

import (
	"sort"
	"strings"
)

type byLen []string

func (b byLen) Len() int           { return len(b) }
func (b byLen) Less(i, j int) bool { return len(b[i]) < len(b[j]) }
func (b byLen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func run() string {
	items := byLen{"longer", "ab", "mediumish", "x", "midd"}
	sort.Sort(items)
	return strings.Join([]string(items), ",")
}
