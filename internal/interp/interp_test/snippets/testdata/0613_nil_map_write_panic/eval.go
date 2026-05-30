package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	var m map[string]int
	m["x"] = 1
	return 0
}
