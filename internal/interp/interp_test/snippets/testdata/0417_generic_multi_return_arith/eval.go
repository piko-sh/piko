package main

func split[T any](xs []T) (T, []T) {
	if len(xs) == 0 {
		var zero T
		return zero, nil
	}
	return xs[0], xs[1:]
}

func run() int {
	head, tail := split([]int{10, 20, 30})
	if len(tail) != 2 {
		return -1
	}
	return head + tail[0] + tail[1]
}
