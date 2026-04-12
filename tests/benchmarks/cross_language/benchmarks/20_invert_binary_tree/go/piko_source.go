package main

import "time"

const treeDepth = 15

const foldMask = 0xFFFFFFFF

const foldMultiplier = 31

type node struct {
	value uint32

	left *node

	right *node
}

func makeTree(depth int, value uint32) *node {
	n := &node{value: value}
	if depth > 0 {
		n.left = makeTree(depth-1, (value*2)&foldMask)
		n.right = makeTree(depth-1, (value*2+1)&foldMask)
	}
	return n
}

func countNodes(t *node) int {
	if t == nil {
		return 0
	}
	return 1 + countNodes(t.left) + countNodes(t.right)
}

func invert(t *node) {
	if t == nil {
		return
	}
	t.left, t.right = t.right, t.left
	invert(t.left)
	invert(t.right)
}

func inorderFold(t *node, state uint32) uint32 {
	if t == nil {
		return state
	}
	state = inorderFold(t.left, state)
	state = (state*foldMultiplier + t.value) & foldMask
	state = inorderFold(t.right, state)
	return state
}

func Run() string {
	return doInvertBinaryTree()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doInvertBinaryTree()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doInvertBinaryTree() string {
	tree := makeTree(treeDepth, 1)
	preCount := countNodes(tree)
	preFold := inorderFold(tree, 0)
	invert(tree)
	postCount := countNodes(tree)
	postFold := inorderFold(tree, 0)
	return intToDecimal(preCount) + "," +
		uintToDecimal(preFold) + "," +
		intToDecimal(postCount) + "," +
		uintToDecimal(postFold)
}

func intToDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := [20]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		digits[position] = '-'
	}
	return string(digits[position:])
}

func uintToDecimal(value uint32) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[position:])
}
