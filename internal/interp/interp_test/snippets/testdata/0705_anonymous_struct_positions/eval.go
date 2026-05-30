package main

import "fmt"

func process(p struct{ X, Y int }) int {
	return p.X*10 + p.Y
}

func makePoint(x, y int) struct{ X, Y int } {
	return struct{ X, Y int }{X: x, Y: y}
}

func run() string {
	result := ""

	result += fmt.Sprintf("paramReturn:%d;", process(makePoint(3, 7)))

	type Pair = struct{ X, Y int }
	a := Pair{X: 1, Y: 2}
	b := Pair{X: 1, Y: 2}
	c := Pair{X: 9, Y: 9}
	result += fmt.Sprintf("eq1:%v;eq2:%v;", a == b, a == c)

	mapOfAnon := map[string]struct{ Tag string }{
		"first":  {Tag: "alpha"},
		"second": {Tag: "beta"},
	}
	result += "mapVal:" + mapOfAnon["first"].Tag + ";"

	sliceOfAnon := []struct {
		Tag   string
		Count int
	}{
		{Tag: "a", Count: 1},
		{Tag: "b", Count: 2},
	}
	for _, item := range sliceOfAnon {
		result += fmt.Sprintf("%s/%d,", item.Tag, item.Count)
	}

	return result
}
