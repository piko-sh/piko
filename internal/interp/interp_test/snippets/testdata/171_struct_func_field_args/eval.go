package main

type Op struct{ Compute func(int, int) int }

func run() int {
	op := Op{Compute: func(a, b int) int { return a + b }}
	return op.Compute(10, 32)
}
