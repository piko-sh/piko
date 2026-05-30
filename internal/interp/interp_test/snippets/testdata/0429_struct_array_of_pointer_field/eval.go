package main

type node struct {
	children [4]*node
	terminal bool
}

func makeNode() *node {
	return &node{}
}

func insert(root *node, word string) {
	n := root
	for i := 0; i < len(word); i++ {
		slot := int(word[i] - 'a')
		if n.children[slot] == nil {
			n.children[slot] = makeNode()
		}
		n = n.children[slot]
	}
	n.terminal = true
}

func run() int {
	root := makeNode()
	insert(root, "ab")
	insert(root, "ac")
	if root.children[0] == nil {
		return 1
	}
	if root.children[0].children[1] == nil {
		return 2
	}
	if !root.children[0].children[1].terminal {
		return 3
	}
	if root.children[0].children[2] == nil {
		return 4
	}
	if !root.children[0].children[2].terminal {
		return 5
	}
	return 0
}
