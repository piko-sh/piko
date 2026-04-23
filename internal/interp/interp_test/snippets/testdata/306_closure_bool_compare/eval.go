package main

import "sort"

type Item struct {
	Name  string
	Score int
}

func run() string {
	items := []Item{
		{"b", 2},
		{"a", 3},
		{"c", 1},
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})
	return items[0].Name + items[1].Name + items[2].Name
}
