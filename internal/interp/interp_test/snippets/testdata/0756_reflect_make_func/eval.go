package main

import (
	"fmt"
	"reflect"
)

func run() string {
	funcType := reflect.FuncOf(
		[]reflect.Type{reflect.TypeOf(0), reflect.TypeOf(0)},
		[]reflect.Type{reflect.TypeOf(0)},
		false,
	)
	impl := func(args []reflect.Value) []reflect.Value {
		sum := args[0].Int() + args[1].Int()
		return []reflect.Value{reflect.ValueOf(int(sum))}
	}
	dynFn := reflect.MakeFunc(funcType, impl)
	out := dynFn.Call([]reflect.Value{reflect.ValueOf(7), reflect.ValueOf(35)})
	return fmt.Sprintf("dyn=%d", out[0].Int())
}
