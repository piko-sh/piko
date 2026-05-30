package main

import "maps"

func run() int {
	dst := map[string]int{"a": 1}
	src := map[string]int{"b": 2, "c": 3}
	maps.Copy(dst, src)
	return dst["a"] + dst["b"] + dst["c"]
}
