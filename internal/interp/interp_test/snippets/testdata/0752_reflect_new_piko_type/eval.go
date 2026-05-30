package main

import (
	"fmt"
	"reflect"
)

type Item struct {
	ID   int
	Name string
}

func run() string {
	t := reflect.TypeOf(Item{})
	ptr := reflect.New(t)
	elem := ptr.Elem()
	elem.FieldByName("ID").SetInt(42)
	elem.FieldByName("Name").SetString("widget")
	item := ptr.Interface().(*Item)
	return fmt.Sprintf("id=%d,name=%s", item.ID, item.Name)
}
