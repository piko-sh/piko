package main

type Node struct {
	V    int
	Next *Node
}

func build(values []int) *Node {
	var head *Node
	for i := len(values) - 1; i >= 0; i-- {
		head = &Node{V: values[i], Next: head}
	}
	return head
}

func sum(n *Node) int {
	total := 0
	for cur := n; cur != nil; cur = cur.Next {
		total += cur.V
	}
	return total
}

func run() int {
	return sum(build([]int{1, 2, 3, 4, 5, 6}))
}
