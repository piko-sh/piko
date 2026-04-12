package main

func listLength[T any](head *Node[T]) int {
	n := 0
	for cur := head; cur != nil; cur = cur.Next {
		n++
	}
	return n
}
