package main

type Stringer interface {
	String() string
}
type Name struct {
	V string
}

func (n Name) String() string {
	return n.V
}
func greet(s Stringer) string {
	return s.String()
}

func run() string {
	n := Name{V: "world"}
	return greet(n)
}
