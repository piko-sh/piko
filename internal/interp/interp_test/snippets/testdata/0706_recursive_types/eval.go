package main

import "fmt"

type Node struct {
	Value int
	Next  *Node
}

type DNode struct {
	Value int
	Prev  *DNode
	Next  *DNode
}

type Tree struct {
	Value int
	Left  *Tree
	Right *Tree
}

func buildList(values []int) *Node {
	var head *Node
	for index := len(values) - 1; index >= 0; index-- {
		head = &Node{Value: values[index], Next: head}
	}
	return head
}

func sumList(head *Node) int {
	total := 0
	for current := head; current != nil; current = current.Next {
		total += current.Value
	}
	return total
}

func buildDoubly(values []int) *DNode {
	var head *DNode
	var tail *DNode
	for _, value := range values {
		node := &DNode{Value: value, Prev: tail}
		if tail != nil {
			tail.Next = node
		} else {
			head = node
		}
		tail = node
	}
	return head
}

func sumDoublyBothWays(head *DNode) (forward int, backward int) {
	var tail *DNode
	for current := head; current != nil; current = current.Next {
		forward += current.Value
		tail = current
	}
	for current := tail; current != nil; current = current.Prev {
		backward += current.Value
	}
	return
}

func inOrder(t *Tree, accumulator *[]int) {
	if t == nil {
		return
	}
	inOrder(t.Left, accumulator)
	*accumulator = append(*accumulator, t.Value)
	inOrder(t.Right, accumulator)
}

func run() string {
	result := ""

	list := buildList([]int{1, 2, 3, 4, 5})
	result += fmt.Sprintf("listSum:%d;", sumList(list))

	doubly := buildDoubly([]int{10, 20, 30, 40})
	forward, backward := sumDoublyBothWays(doubly)
	result += fmt.Sprintf("doubly:%d/%d;", forward, backward)

	tree := &Tree{
		Value: 4,
		Left: &Tree{
			Value: 2,
			Left:  &Tree{Value: 1},
			Right: &Tree{Value: 3},
		},
		Right: &Tree{
			Value: 6,
			Left:  &Tree{Value: 5},
			Right: &Tree{Value: 7},
		},
	}
	var visited []int
	inOrder(tree, &visited)
	result += "tree:"
	for _, v := range visited {
		result += fmt.Sprintf("%d,", v)
	}

	return result
}
