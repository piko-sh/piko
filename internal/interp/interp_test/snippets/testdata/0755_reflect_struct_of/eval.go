package main

import (
	"fmt"
	"reflect"
)

func run() string {
	fields := []reflect.StructField{
		{Name: "A", Type: reflect.TypeOf(0)},
		{Name: "B", Type: reflect.TypeOf("")},
	}
	t := reflect.StructOf(fields)
	v := reflect.New(t).Elem()
	v.FieldByName("A").SetInt(7)
	v.FieldByName("B").SetString("hello")
	return fmt.Sprintf("a=%d,b=%s,kind=%s,fields=%d",
		v.FieldByName("A").Int(),
		v.FieldByName("B").String(),
		t.Kind(),
		t.NumField())
}
