package main

func run() int {
	names := make([]string, 5)
	names[0] = "alice"
	names[1] = "bob"
	names[2] = "carol"
	names[3] = "dave"
	names[4] = "eve"
	target := "carol"
	for i := 0; i < len(names); i++ {
		if names[i] == target {
			return i
		}
	}
	return -1
}
