package main

import "fmt"

type Number interface {
	~int | ~int64 | ~float64
}

type Ordered interface {
	Number | ~string
}

func Sum[T Number](values []T) T {
	var total T
	for _, v := range values {
		total += v
	}
	return total
}

func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

func (p Pair[K, V]) String() string {
	return fmt.Sprintf("%v=%v", p.Key, p.Value)
}

type MyInt int

func run() string {
	result := ""

	result += fmt.Sprintf("sumInts:%d;", Sum[int]([]int{1, 2, 3, 4, 5}))
	result += fmt.Sprintf("sumInt64:%d;", Sum[int64]([]int64{10, 20, 30}))
	result += fmt.Sprintf("sumFloats:%v;", Sum[float64]([]float64{1.5, 2.5, 3.0}))

	result += fmt.Sprintf("sumMyInt:%d;", Sum[MyInt]([]MyInt{1, 2, 3}))

	result += fmt.Sprintf("maxInts:%d;", Max[int](7, 3))
	result += fmt.Sprintf("maxStrings:%s;", Max[string]("apple", "banana"))

	p := Pair[string, int]{Key: "answer", Value: 42}
	result += "pair:" + p.String()

	return result
}
