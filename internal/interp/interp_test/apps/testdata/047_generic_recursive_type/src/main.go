package main

import "fmt"

func entrypoint() string {
	c := &Node[string]{Value: "c"}
	b := &Node[string]{Value: "b", Next: c}
	a := &Node[string]{Value: "a", Next: b}
	var values []string
	for cur := a; cur != nil; cur = cur.Next {
		values = append(values, cur.Value)
	}
	return fmt.Sprintf("values=%v count=%d", values, listLength(a))
}

func main() {
	fmt.Println(entrypoint())
}
