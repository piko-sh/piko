package main

var registry = make(map[string]int)

func init() {
	registry["alpha"] = 1
	registry["beta"] = 2
}

func run() int {
	return registry["alpha"] + registry["beta"]
}
