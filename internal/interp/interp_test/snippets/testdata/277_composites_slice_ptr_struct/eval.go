package main

type Item struct {
	Name string
	Qty  int
}

func run() int {
	items := []*Item{
		{Name: "apple", Qty: 3},
		{Name: "pear", Qty: 7},
		{Name: "plum", Qty: 11},
	}
	total := 0
	for _, it := range items {
		total += it.Qty
	}
	return total
}
