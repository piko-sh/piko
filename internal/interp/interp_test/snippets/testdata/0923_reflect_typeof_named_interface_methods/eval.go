package main

import (
	"fmt"
	"reflect"
	"strings"
)

type triple interface {
	Alpha() int
	Beta() int
	Gamma() int
}

func run() string {
	t := reflect.TypeOf((*triple)(nil)).Elem()
	var sb strings.Builder
	fmt.Fprintf(&sb, "n=%d ", t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		fmt.Fprintf(&sb, "%d:%s ", i, t.Method(i).Name)
	}
	if m, ok := t.MethodByName("Beta"); ok {
		fmt.Fprintf(&sb, "byName=%s ", m.Name)
	}
	if _, ok := t.MethodByName("Missing"); !ok {
		sb.WriteString("missing=ok")
	}
	return sb.String()
}
