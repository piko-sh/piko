package main

import (
	"fmt"
	"reflect"
)

type User struct {
	ID   int    `json:"id" db:"user_id"`
	Name string `json:"name,omitempty"`
	Pass string `json:"-"`
}

func run() string {
	result := ""
	t := reflect.TypeOf(User{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
		dbTag := f.Tag.Get("db")
		result += fmt.Sprintf("%s:json=%q,db=%q;", f.Name, jsonTag, dbTag)
	}
	return result
}
