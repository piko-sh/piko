package main

import (
	"fmt"
	"math/rand"
)

func run() string {
	r := rand.New(rand.NewSource(42))
	result := ""
	result += fmt.Sprintf("intn10=%d,%d,%d;", r.Intn(10), r.Intn(10), r.Intn(10))
	result += fmt.Sprintf("float=%.4f;", r.Float64())
	perm := r.Perm(5)
	result += fmt.Sprintf("perm=%v", perm)
	return result
}
