package main

type Tree struct {
	Name string
	Kids map[string]*Tree
}

func find(t *Tree, path []string) string {
	cur := t
	for _, name := range path {
		next, ok := cur.Kids[name]
		if !ok {
			return ""
		}
		cur = next
	}
	return cur.Name
}

func run() string {
	root := &Tree{
		Name: "root",
		Kids: map[string]*Tree{
			"a": {Name: "a", Kids: map[string]*Tree{
				"a1": {Name: "a1"},
				"a2": {Name: "a2", Kids: map[string]*Tree{"a2a": {Name: "leaf"}}},
			}},
			"b": {Name: "b"},
		},
	}
	return find(root, []string{"a", "a2", "a2a"}) + "/" + find(root, []string{"b"})
}
