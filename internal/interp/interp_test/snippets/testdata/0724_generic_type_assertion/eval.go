package main

import "fmt"

type Box[T any] struct {
	V T
}

func (b Box[T]) Describe() string {
	return fmt.Sprintf("Box{%v}", b.V)
}

func run() string {
	result := ""

	var x any = Box[int]{V: 7}
	if b, ok := x.(Box[int]); ok {
		result += fmt.Sprintf("int=%s;", b.Describe())
	} else {
		result += "int_fail;"
	}

	var y any = Box[string]{V: "hi"}
	if b, ok := y.(Box[string]); ok {
		result += fmt.Sprintf("str=%s;", b.Describe())
	} else {
		result += "str_fail;"
	}

	if _, ok := x.(Box[string]); ok {
		result += "wrong_pos"
	} else {
		result += "wrong_neg"
	}

	return result
}
