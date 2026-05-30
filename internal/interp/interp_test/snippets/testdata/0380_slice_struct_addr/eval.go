package main

type Cell struct {
	V int
}

func run() int {
	cells := []Cell{{V: 1}, {V: 2}, {V: 3}}
	for i := range cells {
		cells[i].V += 100
	}
	return cells[0].V + cells[1].V + cells[2].V
}
