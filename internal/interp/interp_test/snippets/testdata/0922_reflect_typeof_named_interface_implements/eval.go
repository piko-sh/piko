package main

import (
	"fmt"
	"reflect"
)

type reader interface {
	Read() string
}

type readerWriter interface {
	Read() string
	Write(string)
}

func run() string {
	rw := reflect.TypeOf((*readerWriter)(nil)).Elem()
	r := reflect.TypeOf((*reader)(nil)).Elem()
	return fmt.Sprintf("rw.Implements(r)=%v r.Implements(rw)=%v",
		rw.Implements(r), r.Implements(rw))
}
