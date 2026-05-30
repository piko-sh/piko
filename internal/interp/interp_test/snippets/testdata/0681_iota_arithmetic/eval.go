package main

import "fmt"

const (
	FlagA = 1 << iota
	FlagB
	FlagC
	FlagD
	FlagE
)

const (
	StatusZero = iota
	StatusOne
	StatusTwo
)

const (
	A = 10 + iota
	B
	C
)

const (
	X = 1 << (iota + 4)
	Y
	Z
)

const (
	First, Second = iota, iota * 10
	Third, Fourth
	Fifth, Sixth
)

func run() string {
	result := ""
	result += fmt.Sprintf("flags:%d,%d,%d,%d,%d;", FlagA, FlagB, FlagC, FlagD, FlagE)
	result += fmt.Sprintf("status:%d,%d,%d;", StatusZero, StatusOne, StatusTwo)
	result += fmt.Sprintf("plus:%d,%d,%d;", A, B, C)
	result += fmt.Sprintf("shift:%d,%d,%d;", X, Y, Z)
	result += fmt.Sprintf("multi:%d/%d,%d/%d,%d/%d", First, Second, Third, Fourth, Fifth, Sixth)
	return result
}
