package main

import (
	"fmt"
	"reflect"
)

func run() string {
	x := 42
	p := &x
	pp := &p

	v1 := reflect.Indirect(reflect.ValueOf(x))
	v2 := reflect.Indirect(reflect.ValueOf(p))
	v3 := reflect.Indirect(reflect.Indirect(reflect.ValueOf(pp)))

	return fmt.Sprintf("v1=%d,v2=%d,v3=%d", v1.Int(), v2.Int(), v3.Int())
}
