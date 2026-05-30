package main

import (
	"fmt"
	"reflect"
)

type Score int

func run() string {
	result := ""
	scoreType := reflect.TypeOf(Score(0))

	ptr := reflect.PointerTo(scoreType)
	result += fmt.Sprintf("ptr=%s;", ptr.Kind())

	sliceT := reflect.SliceOf(scoreType)
	result += fmt.Sprintf("slice=%s;", sliceT.Kind())

	arrayT := reflect.ArrayOf(3, scoreType)
	result += fmt.Sprintf("array=%s,len=%d;", arrayT.Kind(), arrayT.Len())

	mapT := reflect.MapOf(reflect.TypeOf(""), scoreType)
	result += fmt.Sprintf("map=%s;", mapT.Kind())

	funcT := reflect.FuncOf([]reflect.Type{scoreType}, []reflect.Type{reflect.TypeOf("")}, false)
	result += fmt.Sprintf("func=%s,in=%d,out=%d", funcT.Kind(), funcT.NumIn(), funcT.NumOut())

	return result
}
