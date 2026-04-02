package main

func run() int {
	m := make(map[string]int)
	m["key"] = 99
	return m["key"]
}
