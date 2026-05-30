package main

import (
	"fmt"
	"reflect"
	"strings"
)

type Inner struct {
	A int
	B string
}

type Outer struct {
	X Inner
	Y bool
	Z int
}

func run() string {
	v := reflect.ValueOf(Outer{X: Inner{A: 7, B: "hello"}, Y: true, Z: 42})
	var sb strings.Builder
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		sb.WriteString(fmt.Sprintf("[%d] type=%s kind=%s\n", i, f.Type().String(), f.Kind()))
	}
	return sb.String()
}
