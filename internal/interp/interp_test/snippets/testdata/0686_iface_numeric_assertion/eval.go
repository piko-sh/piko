package main

import "fmt"

type Marker interface {
	IsMarker()
}

type X struct{}

func (X) IsMarker() {}

func run() string {
	result := ""

	var any1 any = int64(42)
	if v, ok := any1.(int64); ok {
		result += fmt.Sprintf("int64ok:%d;", v)
	}
	if _, ok := any1.(int); ok {
		result += "intOk;"
	} else {
		result += "intNo;"
	}

	var any2 any = "hi"
	if v, ok := any2.(string); ok {
		result += "stringOk:" + v + ";"
	}
	if _, ok := any2.(int); ok {
		result += "stringAsIntOk;"
	} else {
		result += "stringAsIntNo;"
	}

	var any3 any = X{}
	if v, ok := any3.(Marker); ok {
		_ = v
		result += "markerOk;"
	}

	var any4 any = 5
	if v, ok := any4.(int); ok {
		result += fmt.Sprintf("intLitOk:%d;", v)
	}

	var any5 any = 3.14
	if v, ok := any5.(float64); ok {
		result += fmt.Sprintf("f64Ok:%v;", v)
	}
	if _, ok := any5.(float32); ok {
		result += "f32Ok;"
	} else {
		result += "f32No;"
	}

	var any6 any = nil
	if _, ok := any6.(Marker); ok {
		result += "nilMarkerOk"
	} else {
		result += "nilMarkerNo"
	}

	return result
}
