package main

func contains[T comparable](xs []T, target T) bool {
	for _, v := range xs {
		if v == target {
			return true
		}
	}
	return false
}

func run() string {
	a := contains([]int{1, 2, 3}, 2)
	b := contains([]string{"x", "y"}, "z")
	if a && !b {
		return "ok"
	}
	return "fail"
}
