package main

type Counter struct {
	N int
}

func run() int {
	counters := map[string]*Counter{
		"a": {N: 10},
		"b": {N: 20},
	}
	counters["a"].N += 5
	counters["b"].N *= 2
	return counters["a"].N + counters["b"].N
}
