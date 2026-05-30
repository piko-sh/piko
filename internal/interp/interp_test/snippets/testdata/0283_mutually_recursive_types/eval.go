package main

type Left struct {
	Name  string
	Right *Right
}

type Right struct {
	Value int
	Left  *Left
}

func walk(l *Left) string {
	if l == nil || l.Right == nil {
		return ""
	}
	return l.Name + ">" + l.Right.Left.Name
}

func run() string {
	l := &Left{Name: "a"}
	r := &Right{Value: 42, Left: l}
	l.Right = r
	return walk(l)
}
