package main

import (
	"fmt"
	"reflect"
)

type myiface interface {
	Foo()
}

func run() string {
	t := reflect.TypeOf((*myiface)(nil))
	if t == nil {
		return "reflect.TypeOf returned nil - BUG"
	}
	e := t.Elem()
	return fmt.Sprintf("Type=%s Elem=%s ElemKind=%s", t.String(), e.String(), e.Kind())
}
