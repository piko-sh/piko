package main

type Tree struct {
	V    int
	Kids []*Tree
}

func sum(t *Tree) int {
	if t == nil {
		return 0
	}
	total := t.V
	for _, k := range t.Kids {
		total += sum(k)
	}
	return total
}

func depth(t *Tree) int {
	if t == nil {
		return 0
	}
	best := 0
	for _, k := range t.Kids {
		if d := depth(k); d > best {
			best = d
		}
	}
	return best + 1
}

func run() int {
	root := &Tree{
		V: 1,
		Kids: []*Tree{
			{V: 2, Kids: []*Tree{{V: 4}, {V: 5}}},
			{V: 3, Kids: []*Tree{{V: 6, Kids: []*Tree{{V: 7}}}}},
		},
	}
	return sum(root)*100 + depth(root)
}
