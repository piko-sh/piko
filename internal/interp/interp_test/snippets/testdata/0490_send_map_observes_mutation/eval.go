package main

func run() int {
	ch := make(chan map[string]int, 1)
	m := map[string]int{"a": 1}
	ch <- m
	m["b"] = 2
	received := <-ch
	return received["b"]
}
