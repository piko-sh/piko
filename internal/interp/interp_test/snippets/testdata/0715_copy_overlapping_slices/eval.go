package main

import "fmt"

func run() string {
	result := ""

	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	copy(a[2:], a[0:])
	result += fmt.Sprintf("shiftRight:%v;", a)

	b := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	copy(b[0:], b[2:])
	result += fmt.Sprintf("shiftLeft:%v;", b)

	c := []byte("abcdefghij")
	copy(c[3:7], c[0:4])
	result += fmt.Sprintf("middle:%q;", string(c))

	d := []byte("ABCDEFGHIJ")
	copy(d[1:5], d[3:7])
	result += fmt.Sprintf("crossLeft:%q;", string(d))

	e := []byte("xyzxyzxyzxyz")
	copy(e[2:8], e[0:6])
	result += fmt.Sprintf("longOverlap:%q;", string(e))

	src := []byte("HELLO")
	dst := []byte("xxxxx")
	n := copy(dst, src)
	result += fmt.Sprintf("disjoint:n=%d,dst=%q", n, string(dst))

	return result
}
