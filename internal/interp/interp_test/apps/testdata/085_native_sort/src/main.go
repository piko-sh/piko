package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"testpkg/lib"
)

func entrypoint() string {
	c := lib.Collection{5, 1, 8, 2, 13, 3}
	sort.Sort(c)
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = strconv.Itoa(v)
	}
	return strings.Join(out, ",")
}

func main() {
	fmt.Println(entrypoint())
}
