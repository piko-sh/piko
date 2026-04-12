package main

import (
	"fmt"
	"reflect"
)

type counter interface {
	Count() int
}

func run() string {
	t := reflect.TypeOf((*counter)(nil))
	e := t.Elem()
	return fmt.Sprintf("ptr.PkgPath=%q elem.PkgPath=%q elem.Name=%q",
		t.PkgPath(), e.PkgPath(), e.Name())
}
