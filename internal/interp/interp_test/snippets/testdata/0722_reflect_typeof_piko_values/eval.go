package main

import (
	"fmt"
	"reflect"
)

type Score int

type Box struct {
	A int
	B string
}

func run() string {
	result := ""

	var s Score = 42
	st := reflect.TypeOf(s)
	result += fmt.Sprintf("score:kind=%s,name=%s;", st.Kind(), st.Name())

	b := Box{A: 1, B: "hi"}
	bt := reflect.TypeOf(b)
	result += fmt.Sprintf("box:kind=%s,name=%s,fields=%d;", bt.Kind(), bt.Name(), bt.NumField())

	var i any = 7
	it := reflect.TypeOf(i)
	result += fmt.Sprintf("any:kind=%s;", it.Kind())

	slice := []int{1, 2, 3}
	slt := reflect.TypeOf(slice)
	result += fmt.Sprintf("slice:kind=%s,elem=%s", slt.Kind(), slt.Elem().Kind())

	return result
}
