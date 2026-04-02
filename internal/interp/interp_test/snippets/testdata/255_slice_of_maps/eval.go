package main

func run() int {
	s := []map[string]int{
		{"x": 1},
		{"x": 2, "y": 3},
	}
	s[1]["z"] = 4
	return s[0]["x"] + s[1]["y"] + s[1]["z"]
}
