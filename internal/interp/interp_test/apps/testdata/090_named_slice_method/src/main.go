package main

import (
	"fmt"
	"strconv"

	"testpkg/lib"
)

func entrypoint() string {
	filters := lib.NewFilters()
	filters = filters.Add(lib.GreaterThan(5))
	filters = filters.Add(lib.LessThan(20))
	filters = filters.Add(lib.NotEqual(13))

	count := 0
	for _, candidate := range []int{1, 5, 6, 10, 13, 17, 20, 21} {
		if filters.Match(candidate) {
			count++
		}
	}
	return strconv.Itoa(count) + " matched"
}

func main() {
	fmt.Println(entrypoint())
}
