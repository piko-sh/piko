package main

import (
	"fmt"
	"reflect"
)

type Calculator struct {
	Acc int
}

func (c *Calculator) Add(x int) int {
	c.Acc += x
	return c.Acc
}

func (c Calculator) Describe() string {
	return fmt.Sprintf("acc=%d", c.Acc)
}

func run() string {
	c := &Calculator{Acc: 10}
	v := reflect.ValueOf(c)

	addMethod := v.MethodByName("Add")
	out := addMethod.Call([]reflect.Value{reflect.ValueOf(7)})

	descMethod := v.MethodByName("Describe")
	out2 := descMethod.Call(nil)

	return fmt.Sprintf("add=%d;desc=%s", out[0].Int(), out2[0].String())
}
