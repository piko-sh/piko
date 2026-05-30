package main

type P struct {
	V int
}

func run() int {
	a := [3]*P{{V: 1}, {V: 2}, {V: 3}}
	return a[0].V + a[1].V + a[2].V
}
