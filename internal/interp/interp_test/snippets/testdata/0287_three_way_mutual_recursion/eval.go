package main

type A struct {
	Tag string
	B   *B
}

type B struct {
	Weight int
	C      *C
}

type C struct {
	Flag bool
	A    *A
}

func label(a *A) string {
	if a == nil || a.B == nil || a.B.C == nil || a.B.C.A == nil {
		return ""
	}
	return a.Tag + "/" + a.B.C.A.Tag
}

func run() string {
	a := &A{Tag: "root"}
	c := &C{Flag: true, A: a}
	b := &B{Weight: 42, C: c}
	a.B = b
	return label(a)
}
