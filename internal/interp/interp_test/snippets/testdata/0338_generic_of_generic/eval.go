package main

type Pair[A, B any] struct {
	First  A
	Second B
}

type Triple[X, Y, Z any] struct {
	Outer Pair[X, Pair[Y, Z]]
}

func run() string {
	t := Triple[string, int, bool]{
		Outer: Pair[string, Pair[int, bool]]{
			First:  "hi",
			Second: Pair[int, bool]{First: 7, Second: true},
		},
	}
	if t.Outer.Second.First == 7 && t.Outer.Second.Second {
		return t.Outer.First
	}
	return "fail"
}
