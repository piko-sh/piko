package main

import "fmt"

type Stringer interface {
	String() string
}

type Box[T any] struct {
	Value T
	Label string
}

func (b Box[T]) String() string {
	return fmt.Sprintf("%s=%v", b.Label, b.Value)
}

type Counter[T comparable] struct {
	counts map[T]int
}

func NewCounter[T comparable]() *Counter[T] {
	return &Counter[T]{counts: make(map[T]int)}
}

func (c *Counter[T]) Add(value T) {
	c.counts[value]++
}

func (c *Counter[T]) Get(value T) int {
	return c.counts[value]
}

func render(s Stringer) string {
	return s.String()
}

func run() string {
	result := ""

	intBox := Box[int]{Value: 42, Label: "int"}
	strBox := Box[string]{Value: "hi", Label: "str"}
	result += render(intBox) + ";"
	result += render(strBox) + ";"

	values := []Stringer{intBox, strBox, Box[float64]{Value: 3.14, Label: "f64"}}
	for _, item := range values {
		result += item.String() + ","
	}

	counter := NewCounter[string]()
	counter.Add("a")
	counter.Add("a")
	counter.Add("b")
	result += fmt.Sprintf("a=%d/b=%d/c=%d", counter.Get("a"), counter.Get("b"), counter.Get("c"))

	return result
}
