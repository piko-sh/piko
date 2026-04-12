package main

import (
	"fmt"
	"reflect"
	"strings"
)

type myiface interface {
	Foo()
}

type concrete struct{}

func (concrete) Foo() {}

func describe(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("%s(%s)", t.String(), t.Kind())
}

func run() string {
	var (
		nilIface         error            = nil
		typedNilPtr      *concrete        = nil
		typedNilIfacePtr *error           = nil
		emptyIfacePtr    *interface{}     = nil
		anonStructPtr    *struct{ X int } = nil
		slice            []int            = nil
		m                map[string]int   = nil
		ch               chan int         = nil
		fn               func()           = nil
	)

	cases := []struct {
		name string
		t    reflect.Type
	}{
		{"reflect.TypeOf(nil)", reflect.TypeOf(nil)},
		{"reflect.TypeOf(error nil)", reflect.TypeOf(nilIface)},
		{"reflect.TypeOf((*concrete)(nil))", reflect.TypeOf((*concrete)(nil))},
		{"reflect.TypeOf((*error)(nil))", reflect.TypeOf((*error)(nil))},
		{"reflect.TypeOf((*fmt.Stringer)(nil))", reflect.TypeOf((*fmt.Stringer)(nil))},
		{"reflect.TypeOf((*interface{})(nil))", reflect.TypeOf((*interface{})(nil))},
		{"reflect.TypeOf(typedNilPtr)", reflect.TypeOf(typedNilPtr)},
		{"reflect.TypeOf(typedNilIfacePtr)", reflect.TypeOf(typedNilIfacePtr)},
		{"reflect.TypeOf(emptyIfacePtr)", reflect.TypeOf(emptyIfacePtr)},
		{"reflect.TypeOf(anonStructPtr)", reflect.TypeOf(anonStructPtr)},
		{"reflect.TypeOf([]int(nil))", reflect.TypeOf([]int(nil))},
		{"reflect.TypeOf(slice)", reflect.TypeOf(slice)},
		{"reflect.TypeOf(map[string]int(nil))", reflect.TypeOf(map[string]int(nil))},
		{"reflect.TypeOf(m)", reflect.TypeOf(m)},
		{"reflect.TypeOf((chan int)(nil))", reflect.TypeOf((chan int)(nil))},
		{"reflect.TypeOf(ch)", reflect.TypeOf(ch)},
		{"reflect.TypeOf((func())(nil))", reflect.TypeOf((func())(nil))},
		{"reflect.TypeOf(fn)", reflect.TypeOf(fn)},
		{"reflect.TypeOf(42)", reflect.TypeOf(42)},
		{"reflect.TypeOf(\"hi\")", reflect.TypeOf("hi")},
		{"reflect.TypeOf(concrete{})", reflect.TypeOf(concrete{})},
	}

	var sb strings.Builder
	for _, c := range cases {
		result := describe(c.t)
		elemResult := ""
		if c.t != nil {
			switch c.t.Kind() {
			case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Chan, reflect.Map:
				func() {
					defer func() {
						if r := recover(); r != nil {
							elemResult = fmt.Sprintf(" Elem-panic:%v", r)
						}
					}()
					elemResult = " Elem=" + describe(c.t.Elem())
				}()
			}
		}
		sb.WriteString(fmt.Sprintf("%-48s = %s%s\n", c.name, result, elemResult))
	}
	return sb.String()
}
