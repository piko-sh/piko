package main

import "fmt"

type Container[T any] interface {
	Get() T
	Set(v T)
}

type IntCell struct {
	v int
}

func (c *IntCell) Get() int  { return c.v }
func (c *IntCell) Set(v int) { c.v = v }

type StringCell struct {
	v string
}

func (c *StringCell) Get() string  { return c.v }
func (c *StringCell) Set(v string) { c.v = v }

func cycle[T any](c Container[T], val T) T {
	c.Set(val)
	return c.Get()
}

func run() string {
	result := ""

	ic := &IntCell{}
	r1 := cycle[int](ic, 99)
	result += fmt.Sprintf("int=%d;", r1)

	sc := &StringCell{}
	r2 := cycle[string](sc, "hello")
	result += fmt.Sprintf("str=%s", r2)

	return result
}
