package main

import (
	"fmt"
	"reflect"
)

type greeter interface {
	Hello() string
	Goodbye() string
}

func run() string {
	t := reflect.TypeOf((*greeter)(nil))
	e := t.Elem()
	return fmt.Sprintf("ptr=%s ptrKind=%s elem=%s elemKind=%s",
		t.String(), t.Kind(), e.String(), e.Kind())
}
