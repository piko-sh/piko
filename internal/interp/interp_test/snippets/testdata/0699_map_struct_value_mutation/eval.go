package main

import "fmt"

type Counter struct {
	Total int
	Name  string
}

func run() string {
	result := ""

	byValue := map[string]Counter{"a": {Total: 5, Name: "alpha"}}
	c := byValue["a"]
	c.Total++
	byValue["a"] = c
	result += fmt.Sprintf("byValue:%d/%s;", byValue["a"].Total, byValue["a"].Name)

	byPointer := map[string]*Counter{"b": {Total: 10, Name: "beta"}}
	byPointer["b"].Total++
	result += fmt.Sprintf("byPointer:%d/%s;", byPointer["b"].Total, byPointer["b"].Name)

	withArray := map[string][3]int{"c": {1, 2, 3}}
	temp := withArray["c"]
	temp[1] = 99
	withArray["c"] = temp
	result += fmt.Sprintf("array:%v;", withArray["c"])

	withSlice := map[string][]int{"d": {1, 2, 3}}
	withSlice["d"] = append(withSlice["d"], 4, 5)
	result += fmt.Sprintf("slice:%v;", withSlice["d"])

	emptyKey := map[string]Counter{}
	missing := emptyKey["zzz"]
	result += fmt.Sprintf("missing:%d/%s", missing.Total, missing.Name)

	return result
}
