package main

import (
	"fmt"
	"sort"
)

type Point struct {
	X, Y int
}

func run() string {
	result := ""

	arrayKey := map[[2]int]string{
		{0, 0}: "origin",
		{1, 0}: "east",
		{0, 1}: "north",
	}
	result += fmt.Sprintf("array_origin=%s;array_east=%s;array_len=%d;",
		arrayKey[[2]int{0, 0}], arrayKey[[2]int{1, 0}], len(arrayKey))

	structKey := map[Point]int{
		{X: 1, Y: 2}: 100,
		{X: 3, Y: 4}: 200,
		{X: 5, Y: 6}: 300,
	}
	keys := make([]int, 0, len(structKey))
	for k := range structKey {
		keys = append(keys, k.X*100+k.Y)
	}
	sort.Ints(keys)
	result += fmt.Sprintf("struct_keys=%v;", keys)

	anyKey := map[any]string{
		1:     "int-1",
		"two": "str-two",
		3.14:  "float-pi",
		true:  "bool",
	}
	result += fmt.Sprintf("any_int=%s;any_str=%s;any_float=%s;any_bool=%s;any_len=%d",
		anyKey[1], anyKey["two"], anyKey[3.14], anyKey[true], len(anyKey))
	return result
}
