package main

import "fmt"

type Describable interface {
	~int | ~string
	Describe() string
}

type Tag int

func (t Tag) Describe() string { return fmt.Sprintf("tag<%d>", int(t)) }

type Label string

func (l Label) Describe() string { return fmt.Sprintf("label<%s>", string(l)) }

func describe[T Describable](v T) string {
	return v.Describe()
}

func run() string {
	result := ""
	result += describe[Tag](Tag(5)) + ";"
	result += describe[Label](Label("hello"))
	return result
}
