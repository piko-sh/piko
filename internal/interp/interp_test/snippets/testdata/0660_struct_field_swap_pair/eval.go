package main

type pair struct {
	left  *node
	right *node
}

type node struct {
	value int
}

func swap(p *pair) {
	p.left, p.right = p.right, p.left
}

func run() int {
	a := &node{value: 1}
	b := &node{value: 2}
	p := &pair{left: a, right: b}

	swap(p)
	swap(p)
	swap(p)

	return p.left.value*10 + p.right.value
}
