package main

import (
	"fmt"
	"sort"
)

type Pair struct {
	Key int
	Tag string
}

type ByKey []Pair

func (b ByKey) Len() int           { return len(b) }
func (b ByKey) Less(i, j int) bool { return b[i].Key < b[j].Key }
func (b ByKey) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func run() string {
	asc := []int{1, 2, 3, 4, 5}
	desc := []int{5, 4, 3, 2, 1}
	mixed := []int{3, 1, 4, 1, 5}

	result := fmt.Sprintf("asc=%t,desc=%t,mixed=%t;",
		sort.IntsAreSorted(asc),
		sort.IntsAreSorted(desc),
		sort.IntsAreSorted(mixed))

	pairs := ByKey{
		{Key: 2, Tag: "a"},
		{Key: 1, Tag: "b"},
		{Key: 2, Tag: "c"},
		{Key: 1, Tag: "d"},
		{Key: 2, Tag: "e"},
	}
	sort.Stable(pairs)
	for _, p := range pairs {
		result += fmt.Sprintf("%d/%s,", p.Key, p.Tag)
	}
	result += fmt.Sprintf(";isSorted=%t", sort.IsSorted(pairs))
	return result
}
