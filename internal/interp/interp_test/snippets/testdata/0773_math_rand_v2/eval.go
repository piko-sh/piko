package main

import (
	"fmt"
	"math/rand/v2"
)

func run() string {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	r := rand.New(rand.NewChaCha8(seed))
	result := ""
	result += fmt.Sprintf("intn=%d,%d,%d;", r.IntN(100), r.IntN(100), r.IntN(100))
	result += fmt.Sprintf("float=%.4f", r.Float64())
	return result
}
