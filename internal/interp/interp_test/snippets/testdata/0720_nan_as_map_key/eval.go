package main

import (
	"fmt"
	"math"
)

func run() string {
	result := ""
	m := map[float64]int{}
	m[1.0] = 100
	m[math.NaN()] = 200
	m[math.NaN()] = 300
	result += fmt.Sprintf("len=%d;", len(m))

	v1, ok1 := m[math.NaN()]
	result += fmt.Sprintf("get_nan:v=%d,ok=%t;", v1, ok1)

	v2, ok2 := m[1.0]
	result += fmt.Sprintf("get_1:v=%d,ok=%t;", v2, ok2)

	count := 0
	for range m {
		count++
	}
	result += fmt.Sprintf("range=%d", count)
	return result
}
