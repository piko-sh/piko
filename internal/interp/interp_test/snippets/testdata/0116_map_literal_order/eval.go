package main

func run() int {
	m := map[string]int{"x": 10, "y": 20, "z": 30}
	return m["x"] + m["y"] + m["z"]
}
