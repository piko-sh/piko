package main

type Stringer interface{ String() string }
type A struct{}

func (A) String() string { return "A" }

type B struct{}

func (B) String() string { return "B" }

func run() string {
	var s Stringer
	s = A{}
	r1 := s.String()
	s = B{}
	r2 := s.String()
	return r1 + r2
}
