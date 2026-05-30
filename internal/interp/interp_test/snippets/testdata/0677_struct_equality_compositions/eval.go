package main

import "fmt"

type Inner struct {
	X, Y int
}

type Outer struct {
	Inner
	Tag string
}

type WithPointer struct {
	Name  string
	Inner *Inner
}

type WithArray struct {
	Coords [3]int
	Label  string
}

type WithIface struct {
	Tag string
	Any any
}

func boolToStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func run() string {
	result := ""

	a := Outer{Inner: Inner{X: 1, Y: 2}, Tag: "a"}
	b := Outer{Inner: Inner{X: 1, Y: 2}, Tag: "a"}
	c := Outer{Inner: Inner{X: 1, Y: 99}, Tag: "a"}
	result += "outer1=" + boolToStr(a == b) + ";"
	result += "outer2=" + boolToStr(a == c) + ";"

	in := &Inner{X: 10, Y: 20}
	p := WithPointer{Name: "p", Inner: in}
	q := WithPointer{Name: "p", Inner: in}
	r := WithPointer{Name: "p", Inner: &Inner{X: 10, Y: 20}}
	result += "ptr1=" + boolToStr(p == q) + ";"
	result += "ptr2=" + boolToStr(p == r) + ";"

	arr1 := WithArray{Coords: [3]int{1, 2, 3}, Label: "abc"}
	arr2 := WithArray{Coords: [3]int{1, 2, 3}, Label: "abc"}
	arr3 := WithArray{Coords: [3]int{1, 2, 9}, Label: "abc"}
	result += "arr1=" + boolToStr(arr1 == arr2) + ";"
	result += "arr2=" + boolToStr(arr1 == arr3) + ";"

	iface1 := WithIface{Tag: "x", Any: nil}
	iface2 := WithIface{Tag: "x", Any: nil}
	iface3 := WithIface{Tag: "x", Any: 5}
	result += "iface1=" + boolToStr(iface1 == iface2) + ";"
	result += "iface2=" + boolToStr(iface1 == iface3)

	_ = fmt.Sprintf
	return result
}
