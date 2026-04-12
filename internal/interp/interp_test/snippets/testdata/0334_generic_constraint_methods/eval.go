package main

type Stringer interface {
	String() string
}

type Named struct {
	Tag string
}

func (n Named) String() string {
	return "<" + n.Tag + ">"
}

func describe[T Stringer](v T) string {
	return v.String()
}

func run() string {
	return describe(Named{Tag: "alpha"})
}
