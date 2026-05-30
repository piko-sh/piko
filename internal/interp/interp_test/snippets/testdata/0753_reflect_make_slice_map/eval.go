package main

import (
	"fmt"
	"reflect"
)

type Tag string

func run() string {
	result := ""

	sliceType := reflect.SliceOf(reflect.TypeOf(Tag("")))
	slice := reflect.MakeSlice(sliceType, 3, 5)
	slice.Index(0).SetString("a")
	slice.Index(1).SetString("b")
	slice.Index(2).SetString("c")
	result += fmt.Sprintf("slice_len=%d,cap=%d;", slice.Len(), slice.Cap())
	for i := 0; i < slice.Len(); i++ {
		result += slice.Index(i).String() + ","
	}
	result += ";"

	mapType := reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(0))
	m := reflect.MakeMap(mapType)
	m.SetMapIndex(reflect.ValueOf("one"), reflect.ValueOf(1))
	m.SetMapIndex(reflect.ValueOf("two"), reflect.ValueOf(2))
	result += fmt.Sprintf("map_len=%d", m.Len())
	return result
}
