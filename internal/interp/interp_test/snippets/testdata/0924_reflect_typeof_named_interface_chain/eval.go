package main

import (
	"fmt"
	"reflect"
)

type widget interface {
	Render() string
}

type gizmo interface {
	Wind() int
}

func run() string {
	tw := reflect.TypeOf((*widget)(nil)).Elem()
	tg := reflect.TypeOf((*gizmo)(nil)).Elem()
	return fmt.Sprintf("widget=%s gizmo=%s same=%v",
		tw.String(), tg.String(), tw == tg)
}
