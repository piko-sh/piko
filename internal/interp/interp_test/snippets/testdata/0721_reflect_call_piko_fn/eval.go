package main

import (
	"fmt"
	"reflect"
)

func addInts(a, b int) int { return a + b }

func greet(name string) string { return "hello " + name }

func multiReturn(x int) (int, string) {
	return x * 2, fmt.Sprintf("seen %d", x)
}

func run() string {
	result := ""

	fv := reflect.ValueOf(addInts)
	out := fv.Call([]reflect.Value{reflect.ValueOf(7), reflect.ValueOf(5)})
	result += fmt.Sprintf("add=%d;", out[0].Int())

	gv := reflect.ValueOf(greet)
	gout := gv.Call([]reflect.Value{reflect.ValueOf("piko")})
	result += fmt.Sprintf("greet=%s;", gout[0].String())

	mv := reflect.ValueOf(multiReturn)
	mout := mv.Call([]reflect.Value{reflect.ValueOf(3)})
	result += fmt.Sprintf("multi=%d/%s", mout[0].Int(), mout[1].String())

	return result
}
