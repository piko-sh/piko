package main

import (
	"fmt"
	"reflect"
)

type Point struct {
	X, Y int
}

type Tree struct {
	Label string
	Kids  []*Tree
}

func run() string {
	result := ""

	a := Point{X: 1, Y: 2}
	b := Point{X: 1, Y: 2}
	c := Point{X: 1, Y: 3}
	result += fmt.Sprintf("eq=%t;neq=%t;", reflect.DeepEqual(a, b), reflect.DeepEqual(a, c))

	t1 := &Tree{Label: "root", Kids: []*Tree{{Label: "a"}, {Label: "b"}}}
	t2 := &Tree{Label: "root", Kids: []*Tree{{Label: "a"}, {Label: "b"}}}
	t3 := &Tree{Label: "root", Kids: []*Tree{{Label: "a"}, {Label: "c"}}}
	result += fmt.Sprintf("tree_eq=%t;tree_neq=%t;", reflect.DeepEqual(t1, t2), reflect.DeepEqual(t1, t3))

	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 2, "a": 1}
	result += fmt.Sprintf("map_eq=%t", reflect.DeepEqual(m1, m2))

	return result
}
