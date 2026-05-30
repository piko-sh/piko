package main

import (
	"fmt"
	"reflect"
)

type myiface interface {
	Foo()
}

func run() string {
	user := reflect.TypeOf((*myiface)(nil)).Elem()
	std := reflect.TypeOf((*error)(nil)).Elem()
	return fmt.Sprintf("user=%s userKind=%s std=%s stdKind=%s",
		user.String(), user.Kind(), std.String(), std.Kind())
}
