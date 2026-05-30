package main

import "fmt"

func run() string {
	result := ""

	a := uint8(0xF0)
	notA := ^a
	result += fmt.Sprintf("not0xF0=%d;", notA)

	b := uint16(0)
	notB := ^b
	result += fmt.Sprintf("notZero16=%d;", notB)

	var c int8 = 0
	notC := ^c
	result += fmt.Sprintf("notInt8Zero=%d;", notC)

	d := uint32(0xFFFFFFFF)
	notD := ^d
	result += fmt.Sprintf("notAllOnes=%d;", notD)

	e := 5
	twoOps := ^e & 0xFF
	result += fmt.Sprintf("notAnd=%d", twoOps)

	return result
}
