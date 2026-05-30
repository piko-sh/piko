package main

import (
	"fmt"
	"reflect"
)

type Person struct {
	Name string
	Age  int
}

func run() string {
	p := &Person{Name: "alice", Age: 30}
	v := reflect.ValueOf(p).Elem()

	v.FieldByName("Name").SetString("bob")
	v.FieldByName("Age").SetInt(99)

	return fmt.Sprintf("name=%s,age=%d", p.Name, p.Age)
}
