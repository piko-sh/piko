package main

type Node struct {
	Val int
	Tag string
}

func mutate(n *Node) {
	n.Val *= 2
	n.Tag = "mutated"
}

func run() string {
	n := &Node{Val: 10, Tag: "orig"}
	mutate(n)
	return n.Tag
}
