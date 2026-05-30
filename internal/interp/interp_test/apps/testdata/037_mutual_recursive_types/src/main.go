package main

import "fmt"

func entrypoint() string {
	root := &Tree{Name: "trunk"}
	a := &Branch{Label: "a", Parent: root}
	b := &Branch{Label: "b", Parent: root}
	root.Children = []*Branch{a, b}
	return fmt.Sprintf("root=%s children=[%s %s] aParent=%s",
		root.Name, root.Children[0].Label, root.Children[1].Label, a.Parent.Name)
}

func main() {
	fmt.Println(entrypoint())
}
